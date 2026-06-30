# Accessing Spark UI via OpenShift Routes

This guide explains how to access the Spark Application UI using OpenShift Routes with HTTPS for production environments.

## Table of Contents
- [When to Use OpenShift Routes](#when-to-use-openshift-routes)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration Overview](#configuration-overview)
- [Step-by-Step Setup](#step-by-step-setup)
- [Understanding the Configuration](#understanding-the-configuration)
- [Accessing the Spark UI](#accessing-the-spark-ui)
- [TLS/HTTPS Configuration](#tlshttps-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Troubleshooting](#troubleshooting)
- [Comparison with Port-Forward](#comparison-with-port-forward)

---

## When to Use OpenShift Routes

OpenShift Routes are ideal for:

- **Production workloads** - Stable, persistent access to Spark UI
- **Team collaboration** - Shareable URLs accessible by multiple users
- **Long-running monitoring** - Continuous access without active terminal sessions
- **External access** - Access from outside the cluster without VPN
- **Automated tooling** - Integration with monitoring and alerting systems
- **HTTPS security** - Built-in TLS termination for secure access

**Use Routes instead of port-forward when:**
- You need to share UI access with team members
- The Spark application runs for extended periods
- You require HTTPS/TLS security
- Multiple users need simultaneous access
- You're deploying to production environments

---

## Prerequisites

Before you begin, ensure you have:

1. **OpenShift Cluster** - Access to an OpenShift 4.x cluster
2. **Spark Operator** - Kubeflow Spark Operator installed and running
3. **Cluster Domain** - Know your cluster's apps domain (e.g., `apps.cluster.example.com`)
4. **Permissions** - Ability to create SparkApplications in your namespace

### Required Permissions

Your service account needs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: spark-ui-route-access
rules:
- apiGroups: ["sparkoperator.k8s.io"]
  resources: ["sparkapplications"]
  verbs: ["create", "get", "list", "watch", "update", "patch"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses"]
  verbs: ["get", "list"]
- apiGroups: ["route.openshift.io"]
  resources: ["routes"]
  verbs: ["get", "list"]
```

---

## Quick Start

For those who want to get started immediately:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi
  namespace: my-namespace
spec:
  type: Scala
  mode: cluster
  image: quay.io/ssankepe/spark-openshift:3.5.7
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples_2.12-3.5.7.jar
  sparkVersion: "3.5.7"
  restartPolicy:
    type: Never
  driver:
    cores: 1
    memory: "1000m"
    serviceAccount: spark-operator-spark
  executor:
    cores: 1
    instances: 1
    memory: "1000m"
  
  # IMPORTANT: Both configurations required due to operator bug
  sparkUIOptions:
    ingressAnnotations:
      route.openshift.io/termination: "edge"
  
  driverIngressOptions:
    - servicePort: 4040
      servicePortName: "spark-driver-ui-port"
      ingressURLFormat: "spark-pi-{{$appNamespace}}.apps.<your-cluster-domain>"
```

Replace `<your-cluster-domain>` with your cluster's apps domain.

---

## Configuration Overview

To enable Spark UI access via Routes, you need to configure two sections in your SparkApplication:

1. **`sparkUIOptions`** - Contains TLS annotations (workaround for operator bug)
2. **`driverIngressOptions`** - Defines Ingress/Route configuration

### Architecture Flow

```
SparkApplication YAML
        ↓
Spark Operator creates:
  - Driver Pod (Spark UI on port 4040)
  - Service (exposes port 4040)
  - Ingress (with Route annotations)
        ↓
OpenShift Ingress Controller
  - Detects route.openshift.io annotations
  - Creates Route automatically
  - Configures TLS edge termination
        ↓
User accesses:
  https://spark-pi-my-namespace.apps.cluster.example.com
        ↓
OpenShift Router (HAProxy)
  - Terminates TLS
  - Forwards HTTP to Service
        ↓
Service → Driver Pod → Spark UI
```

---

## Step-by-Step Setup

### Step 1: Determine Your Cluster Domain

Find your OpenShift cluster's apps domain:

```bash
# Method 1: Check Ingress config
oc get ingress.config.openshift.io cluster -o jsonpath='{.spec.domain}'

# Method 2: Check existing routes
oc get routes -A | head -5

# Example output:
# apps.rosa.example.mwon.p3.openshiftapps.com
```

### Step 2: Create SparkApplication with Route Configuration

Create a file `spark-pi-with-route.yaml`:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi
  namespace: redhat-ods-applications
spec:
  type: Scala
  mode: cluster
  image: quay.io/ssankepe/spark-openshift:3.5.7
  imagePullPolicy: Always
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples_2.12-3.5.7.jar
  arguments:
    - "1000"
  sparkVersion: "3.5.7"
  restartPolicy:
    type: Never
  
  # Driver configuration
  driver:
    cores: 1
    memory: "1000m"
    serviceAccount: spark-operator-spark
    labels:
      version: "3.5.7"
  
  # Executor configuration
  executor:
    cores: 1
    instances: 1
    memory: "1000m"
    labels:
      version: "3.5.7"
  
  # UI Access Configuration
  # WORKAROUND: Due to operator bug, annotations must be in sparkUIOptions
  sparkUIOptions:
    ingressAnnotations:
      route.openshift.io/termination: "edge"
  
  # Ingress/Route configuration
  driverIngressOptions:
    - servicePort: 4040
      servicePortName: "spark-driver-ui-port"
      ingressURLFormat: "spark-pi-{{$appNamespace}}.apps.rosa.example.mwon.p3.openshiftapps.com"
```

### Step 3: Deploy the SparkApplication

```bash
oc apply -f spark-pi-with-route.yaml
```

### Step 4: Verify Resources Created

```bash
# Check SparkApplication status
oc get sparkapplication spark-pi -n redhat-ods-applications

# Check Service created
oc get svc -n redhat-ods-applications | grep spark-pi

# Check Ingress created
oc get ingress -n redhat-ods-applications | grep spark-pi

# Check Route created (auto-generated from Ingress)
oc get route -n redhat-ods-applications | grep spark-pi
```

### Step 5: Access the Spark UI

Get the Route URL:

```bash
oc get route -n redhat-ods-applications -o jsonpath='{.items[?(@.metadata.labels.sparkoperator\.k8s\.io/app-name=="spark-pi")].spec.host}'
```

Open in browser:
```
https://spark-pi-redhat-ods-applications.apps.rosa.example.mwon.p3.openshiftapps.com
```

---

## Understanding the Configuration

### sparkUIOptions vs driverIngressOptions

The SparkApplication spec has two related but distinct configuration sections:

| Field | Purpose | Creates Service | Creates Ingress |
|-------|---------|----------------|-----------------|
| `sparkUIOptions` | Basic UI service configuration | ✅ Yes (`<app>-ui-svc`) | ❌ No |
| `driverIngressOptions` | Ingress/Route configuration | ✅ Yes (`<app>-driver-<port>`) | ✅ Yes |

**Important:** Due to an operator bug, you must specify TLS annotations in `sparkUIOptions.ingressAnnotations` even though the Ingress is created by `driverIngressOptions`.

### Required Fields

#### sparkUIOptions

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "edge"  # REQUIRED for HTTPS
```

**Why this is required:**
- Modern browsers auto-upgrade HTTP to HTTPS
- Without TLS termination, browser requests fail with 503 errors
- The operator currently only reads annotations from this location (bug)

#### driverIngressOptions

```yaml
driverIngressOptions:
  - servicePort: 4040                                    # REQUIRED
    servicePortName: "spark-driver-ui-port"             # REQUIRED
    ingressURLFormat: "spark-{{$appNamespace}}.apps..." # REQUIRED
```

**Field descriptions:**

- **servicePort**: Port on which Spark UI listens (always 4040 for Spark UI)
- **servicePortName**: Name for the service port (must be non-empty)
- **ingressURLFormat**: URL template for the Route
  - `{{$appName}}` - Replaced with SparkApplication name
  - `{{$appNamespace}}` - Replaced with namespace
  - `{{$appId}}` - Replaced with unique app ID

### URL Format Examples

```yaml
# Simple format with app name
ingressURLFormat: "spark-{{$appName}}.apps.cluster.example.com"
# Result: https://spark-pi.apps.cluster.example.com

# With namespace for multi-tenancy
ingressURLFormat: "{{$appName}}-{{$appNamespace}}.apps.cluster.example.com"
# Result: https://spark-pi-my-namespace.apps.cluster.example.com

# Custom subdomain
ingressURLFormat: "ui-{{$appName}}.spark.apps.cluster.example.com"
# Result: https://ui-spark-pi.spark.apps.cluster.example.com
```

---

## Accessing the Spark UI

### Finding the Route URL

**Method 1: Via oc get route**
```bash
oc get route -n <namespace> -l sparkoperator.k8s.io/app-name=<app-name>
```

**Method 2: Via SparkApplication status** (if supported)
```bash
oc get sparkapplication <app-name> -n <namespace> -o jsonpath='{.status.sparkUIURL}'
```

**Method 3: Describe Ingress**
```bash
oc describe ingress -n <namespace> | grep -A 5 spark-
```

### Spark UI Features

Once accessed, the Spark UI provides:

| Tab | Information | Use Case |
|-----|-------------|----------|
| **Jobs** | Job execution status and timeline | Track overall progress |
| **Stages** | Stage details, tasks, and data | Identify bottlenecks |
| **Storage** | RDD/DataFrame persistence | Monitor cache usage |
| **Environment** | Spark and JVM configuration | Debug configuration issues |
| **Executors** | Executor metrics and logs | Troubleshoot resource issues |
| **SQL** | Query plans and execution | Optimize Spark SQL queries |
| **Streaming** | Streaming statistics (if applicable) | Monitor streaming jobs |

---

## TLS/HTTPS Configuration

### Why HTTPS is Mandatory

Modern browsers (Chrome, Firefox, Safari, Edge) automatically upgrade HTTP connections to HTTPS for security. Without TLS configuration:

- HTTP Route URL: `http://spark-pi.apps.cluster.example.com`
- Browser automatically tries: `https://spark-pi.apps.cluster.example.com`
- Result: **503 Service Unavailable** (TLS not configured)

### TLS Termination Options

OpenShift Routes support several TLS termination modes:

#### 1. Edge Termination (Recommended)

TLS is terminated at the router, traffic forwarded as HTTP to the pod.

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "edge"
```

**Pros:**
- ✅ Simple configuration
- ✅ No certificate management in pods
- ✅ Router handles TLS
- ✅ Works with browser auto-upgrade

**Cons:**
- ⚠️ Traffic between router and pod is unencrypted (within cluster)

#### 2. Passthrough Termination

TLS is terminated at the pod (Spark UI must handle TLS).

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "passthrough"
```

**Pros:**
- ✅ End-to-end encryption

**Cons:**
- ❌ Spark UI doesn't natively support HTTPS
- ❌ Requires sidecar proxy (e.g., nginx, envoy)
- ❌ Complex configuration

#### 3. Re-encrypt Termination

TLS terminated at router, re-encrypted to pod.

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "reencrypt"
```

**Requires:**
- Pod must serve HTTPS
- Certificates configured in pod

**Recommendation:** Use **edge termination** for Spark UI as it's the simplest and most compatible approach.

### Custom TLS Certificates

To use custom certificates instead of OpenShift's default:

```yaml
driverIngressOptions:
  - servicePort: 4040
    servicePortName: "spark-driver-ui-port"
    ingressURLFormat: "spark-{{$appName}}.apps.cluster.example.com"
    ingressTLS:
      - hosts:
          - "spark-{{$appName}}.apps.cluster.example.com"
        secretName: spark-ui-tls-cert
```

Create the TLS secret:

```bash
oc create secret tls spark-ui-tls-cert \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  -n <namespace>
```

---

## Advanced Configuration

### Multiple Ingress Endpoints

You can expose multiple endpoints from the driver:

```yaml
driverIngressOptions:
  # Spark UI
  - servicePort: 4040
    servicePortName: "spark-driver-ui-port"
    ingressURLFormat: "spark-ui-{{$appName}}.apps.cluster.example.com"
  
  # Block Manager UI
  - servicePort: 4041
    servicePortName: "spark-blockmgr-port"
    ingressURLFormat: "spark-blockmgr-{{$appName}}.apps.cluster.example.com"
```

### Custom Annotations

Add additional Route annotations for advanced features:

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "edge"
    haproxy.router.openshift.io/timeout: "5m"
    haproxy.router.openshift.io/rate-limit-connections: "true"
    haproxy.router.openshift.io/rate-limit-connections.concurrent-tcp: "100"
```

**Common annotations:**

| Annotation | Purpose | Example |
|------------|---------|---------|
| `route.openshift.io/termination` | TLS termination type | `"edge"`, `"passthrough"`, `"reencrypt"` |
| `haproxy.router.openshift.io/timeout` | Connection timeout | `"5m"`, `"30s"` |
| `haproxy.router.openshift.io/balance` | Load balancing algorithm | `"roundrobin"`, `"leastconn"` |
| `haproxy.router.openshift.io/disable_cookies` | Disable sticky sessions | `"true"`, `"false"` |

### Ingress Class

Specify a custom ingress class if you have multiple ingress controllers:

```yaml
driverIngressOptions:
  - servicePort: 4040
    servicePortName: "spark-driver-ui-port"
    ingressURLFormat: "spark-{{$appName}}.apps.cluster.example.com"
    ingressClassName: "openshift-default"
```

---

## Troubleshooting

### 503 Service Unavailable

**Symptom:** Browser shows "Application is not available"

**Possible Causes:**

1. **Missing TLS termination annotation**
   ```bash
   # Check if annotation is present
   oc get ingress <ingress-name> -n <namespace> -o yaml | grep termination
   ```
   
   **Solution:** Add `route.openshift.io/termination: "edge"` to `sparkUIOptions.ingressAnnotations`

2. **Driver pod not running**
   ```bash
   # Check driver pod status
   oc get pods -n <namespace> -l spark-role=driver,sparkoperator.k8s.io/app-name=<app-name>
   ```
   
   **Solution:** Wait for driver pod to reach `Running` state

3. **Service not created**
   ```bash
   # Check if service exists
   oc get svc -n <namespace> | grep <app-name>
   ```
   
   **Solution:** Verify SparkApplication has correct `driverIngressOptions` configuration

---

### Route Not Created

**Symptom:** `oc get route` shows no route for the application

**Possible Causes:**

1. **Ingress not created**
   ```bash
   # Check ingress
   oc get ingress -n <namespace> | grep <app-name>
   ```
   
   **Solution:** Verify `driverIngressOptions` is configured correctly

2. **OpenShift Ingress Controller not running**
   ```bash
   # Check ingress operator
   oc get pods -n openshift-ingress-operator
   oc get pods -n openshift-ingress
   ```
   
   **Solution:** Contact cluster administrator

---

### Wrong URL Format

**Symptom:** Route created but hostname is incorrect

**Check Route hostname:**
```bash
oc get route <route-name> -n <namespace> -o jsonpath='{.spec.host}'
```

**Solution:** Update `ingressURLFormat` in `driverIngressOptions`:

```yaml
driverIngressOptions:
  - ingressURLFormat: "spark-{{$appName}}-{{$appNamespace}}.apps.correct-domain.com"
```

---

### Certificate Errors

**Symptom:** Browser shows "Your connection is not private" or SSL certificate warnings

**Possible Causes:**

1. **Using OpenShift's default wildcard certificate**
   - This is normal for development clusters
   - Accept the certificate warning (not recommended for production)

2. **Custom certificate mismatch**
   ```bash
   # Check certificate in Route
   oc get route <route-name> -n <namespace> -o yaml | grep -A 10 tls
   ```
   
   **Solution:** Ensure certificate SAN includes the Route hostname

---

### Annotations Not Applied

**Symptom:** Ingress created without expected annotations

**This is a known operator bug.** Annotations specified in `driverIngressOptions[].ingressAnnotations` are ignored.

**Workaround:** Place annotations in `sparkUIOptions.ingressAnnotations`:

```yaml
sparkUIOptions:
  ingressAnnotations:
    route.openshift.io/termination: "edge"
    # Add other annotations here
```

**Permanent Fix:** Track upstream bug report (link to be added)

---

### Cannot Access from Outside Cluster

**Symptom:** Route works from within cluster but not externally

**Check:**
```bash
# Test from outside cluster
curl -I https://<route-hostname>

# Check Route status
oc get route <route-name> -n <namespace> -o yaml
```

**Possible Causes:**

1. **Firewall blocking port 443**
   - Contact network administrator

2. **DNS not resolving**
   ```bash
   nslookup <route-hostname>
   ```
   - Ensure DNS points to cluster's router IP

3. **Router not exposed externally**
   ```bash
   oc get svc -n openshift-ingress
   ```
   - Check if router service has external IP/LoadBalancer

---

## Comparison with Port-Forward

| Feature | Port-Forward | OpenShift Route |
|---------|-------------|-----------------|
| **Setup** | ⭐ Simple | ⭐⭐⭐ Requires YAML config |
| **Access** | 🏠 Local only | 🌍 Public/Team access |
| **Persistence** | ❌ Terminal session | ✅ Persistent |
| **HTTPS** | ❌ HTTP only | ✅ TLS edge termination |
| **Authentication** | ✅ Cluster credentials | ⚠️ Optional (can add) |
| **Production** | ❌ Not recommended | ✅ Production-ready |
| **Shareability** | ❌ Cannot share | ✅ Shareable URL |
| **Automation** | ❌ Manual | ✅ Automated |

### Decision Matrix

**Use Port-Forward when:**
- 🛠️ Quick debugging during development
- 👤 Single user access
- ⏱️ Short-lived access (< 1 hour)
- 🔒 Strong authentication required

**Use Routes when:**
- 🚀 Production deployments
- 👥 Team collaboration
- ⏰ Long-running jobs
- 🌐 External stakeholder access
- 🔐 HTTPS security required

---

## Examples

### Example 1: Production Spark Job with Route

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: data-processing-job
  namespace: data-platform
spec:
  type: Scala
  mode: cluster
  image: quay.io/myorg/spark:3.5.7
  mainClass: com.example.DataProcessor
  mainApplicationFile: s3a://bucket/app.jar
  sparkVersion: "3.5.7"
  
  driver:
    cores: 2
    memory: "4000m"
    serviceAccount: spark-operator-spark
  
  executor:
    cores: 4
    instances: 10
    memory: "8000m"
  
  sparkUIOptions:
    ingressAnnotations:
      route.openshift.io/termination: "edge"
      haproxy.router.openshift.io/timeout: "10m"
  
  driverIngressOptions:
    - servicePort: 4040
      servicePortName: "spark-driver-ui-port"
      ingressURLFormat: "data-processing-{{$appNamespace}}.apps.prod.example.com"
```

### Example 2: Development with Custom Domain

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: ml-training
  namespace: ml-dev
spec:
  type: Python
  mode: cluster
  image: quay.io/myorg/spark-ml:latest
  mainApplicationFile: local:///opt/app/train.py
  sparkVersion: "3.5.7"
  
  driver:
    cores: 1
    memory: "2000m"
    serviceAccount: spark-operator-spark
  
  executor:
    cores: 2
    instances: 5
    memory: "4000m"
  
  sparkUIOptions:
    ingressAnnotations:
      route.openshift.io/termination: "edge"
  
  driverIngressOptions:
    - servicePort: 4040
      servicePortName: "spark-driver-ui-port"
      ingressURLFormat: "ml-training-ui.dev.apps.cluster.example.com"
```

---

## Next Steps

After successfully accessing Spark UI via Routes:

1. **For Completed Jobs:** Set up [Spark History Server](spark-history-server-setup.md) to access UIs of finished applications

2. **For Monitoring:** Integrate Spark metrics with Prometheus and Grafana

3. **For Security:** Add OAuth authentication to Routes:
   ```yaml
   sparkUIOptions:
     ingressAnnotations:
       route.openshift.io/termination: "edge"
       haproxy.router.openshift.io/auth-type: "oauth"
   ```

4. **For Development:** Use [Port-Forward](SparkUI-PortForward.md) for quick testing

---

## Related Resources

- [Port-Forward Guide](SparkUI-PortForward.md) - Alternative access method for development
- [Spark UI Documentation](https://spark.apache.org/docs/latest/web-ui.html)
- [OpenShift Routes](https://docs.redhat.com/en/documentation/openshift_container_platform/4.11/html/networking/configuring-routes)
- [Kubeflow Spark Operator](https://github.com/kubeflow/spark-operator)
- [Spark Operator API Docs](../../docs/api-docs.md)

---

**Document Version:** 1.0  
**Last Updated:** May 28, 2026  
**JIRA:** RHOAIENG-60635  
**Maintained By:** RHOAI Engineering Team
