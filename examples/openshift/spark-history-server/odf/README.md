# Spark History Server with OpenShift Data Foundation (ODF)

Complete guide to setting up Spark History Server with OpenShift Data Foundation on OpenShift. ODF provides Ceph-based storage with both S3-compatible object storage and ReadWriteMany PVCs.

## Table of Contents
- [What is OpenShift Data Foundation](#what-is-openshift-data-foundation)
- [Why ODF for Spark History Server](#why-odf-for-spark-history-server)
- [Prerequisites](#prerequisites)
- [Two Approaches](#two-approaches)
- [Approach 1: ODF S3-Compatible Storage (Recommended)](#approach-1-odf-s3-compatible-storage-recommended)
- [Approach 2: ODF RWX PVC Storage](#approach-2-odf-rwx-pvc-storage)
- [Troubleshooting](#troubleshooting)
- [Advanced Configuration](#advanced-configuration)

---

## What is OpenShift Data Foundation

OpenShift Data Foundation (ODF) is Red Hat's software-defined storage solution built on Ceph. It provides:

- **Object Storage (S3-compatible)** via NooBaa/RADOS Gateway (RGW)
- **Block Storage** (RWO PVCs)
- **File Storage** (RWX PVCs) via CephFS

For Spark History Server, we use either:
1. **S3-compatible object storage** (via ObjectBucketClaim)
2. **ReadWriteMany PVCs** (via CephFS)

---

## Why ODF for Spark History Server

**Advantages:**

✅ **Enterprise Support** - Fully supported by Red Hat with enterprise SLAs  
✅ **Self-Contained** - Runs entirely within your OpenShift cluster, no external services  
✅ **Air-Gap Ready** - Works in disconnected/air-gapped environments  
✅ **Predictable Costs** - No per-GB cloud storage fees, pay for cluster resources only  
✅ **Flexible** - Provides both S3 and file storage from same system  
✅ **Production-Ready** - Field-proven, enterprise-grade storage solution  

**When to use ODF:**
- ✅ Production OpenShift deployments (on-premises or cloud)
- ✅ Disconnected/air-gapped environments
- ✅ Multi-tenant clusters requiring consistent storage across teams
- ✅ Environments where external cloud storage is not available

---

## Prerequisites

### 1. OpenShift Cluster with ODF Installed

Verify ODF is installed and healthy:

```bash
# Check ODF operator
oc get csv -n openshift-storage | grep odf-operator

# Check storage cluster
oc get storagecluster -n openshift-storage

# Should show: NAME                 AGE   PHASE   EXTERNAL   CREATED AT             VERSION
#              ocs-storagecluster   XXd   Ready   false      YYYY-MM-DDTHH:MM:SSZ   X.X.X
```

If ODF is not installed, see [Red Hat OpenShift Data Foundation documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/).

### 2. Spark Operator Installed

```bash
oc get pods -n spark-operator
# Should show spark-operator-controller and spark-operator-webhook pods running
```

### 3. Command-Line Tools

- `oc` CLI installed and configured
- Access to create resources in your namespace

---

## Two Approaches

ODF supports both S3-compatible object storage and RWX PVCs. Choose based on your needs:

| Approach | Best For | Complexity |
|----------|----------|------------|
| **S3-Compatible (ObjectBucketClaim)** | Production, scalability, multi-namespace | Medium |
| **RWX PVC (CephFS)** | Simple setup, single namespace | Low |

**Recommendation:** Use **S3-compatible approach** for production deployments.

---

## Approach 1: ODF S3-Compatible Storage (Recommended)

This approach uses ODF's NooBaa to provide S3-compatible object storage.

### Step 1: Create ObjectBucketClaim

ObjectBucketClaim automatically provisions an S3 bucket and credentials.

Create `odf-bucket.yaml`:

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name: spark-event-logs
  namespace: spark-operator
spec:
  generateBucketName: spark-logs
  storageClassName: openshift-storage.noobaa.io
```

Apply:

```bash
oc apply -f odf-bucket.yaml
```

### Step 2: Verify Bucket and Credentials Created

```bash
# Check ObjectBucketClaim status
oc get objectbucketclaim spark-event-logs -n spark-operator

# Should show: NAME                 STORAGE-CLASS                  PHASE   AGE
#              spark-event-logs     openshift-storage.noobaa.io    Bound   XXs

# ODF automatically creates:
# 1. Secret with S3 credentials
oc get secret spark-event-logs -n spark-operator

# 2. ConfigMap with bucket details
oc get configmap spark-event-logs -n spark-operator
```

### Step 3: Get ODF S3 Configuration

Extract the bucket name and endpoint:

```bash
# Get bucket name
BUCKET_NAME=$(oc get configmap spark-event-logs -n spark-operator \
  -o jsonpath='{.data.BUCKET_NAME}')

# Get S3 endpoint
S3_ENDPOINT=$(oc get configmap spark-event-logs -n spark-operator \
  -o jsonpath='{.data.BUCKET_HOST}')

# Get S3 port (usually 443 for HTTPS or 80 for HTTP)
S3_PORT=$(oc get configmap spark-event-logs -n spark-operator \
  -o jsonpath='{.data.BUCKET_PORT}')

echo "Bucket: $BUCKET_NAME"
echo "Endpoint: $S3_ENDPOINT"
echo "Port: $S3_PORT"
```

**Note:** ODF creates bucket names like `spark-logs-12345abcde-6789f-1234-5678-90abcdef1234`.

### Step 4: Configure SparkApplication for Event Logging

Create `spark-pi-with-odf.yaml`:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi-odf
  namespace: spark-operator
spec:
  type: Scala
  mode: cluster
  image: quay.io/opendatahub/spark:3.5.7
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples_2.12-3.5.7.jar
  arguments: ["1000"]
  sparkVersion: "3.5.7"
  
  restartPolicy:
    type: Never
  
  # Event logging configuration for ODF
  sparkConf:
    # Enable event logging
    "spark.eventLog.enabled": "true"
    "spark.eventLog.dir": "s3a://BUCKET_NAME/spark-event-logs"  # Replace BUCKET_NAME
    "spark.eventLog.compress": "true"
    
    # ODF S3 configuration
    "spark.hadoop.fs.s3a.endpoint": "S3_ENDPOINT:S3_PORT"  # Replace with actual values
    "spark.hadoop.fs.s3a.impl": "org.apache.hadoop.fs.s3a.S3AFileSystem"
    "spark.hadoop.fs.s3a.path.style.access": "true"  # Required for ODF NooBaa
    "spark.hadoop.fs.s3a.connection.ssl.enabled": "true"  # Use false if HTTP
  
  driver:
    cores: 1
    memory: "1000m"
    serviceAccount: spark-operator-spark
    env:
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: spark-event-logs
          key: AWS_ACCESS_KEY_ID
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: spark-event-logs
          key: AWS_SECRET_ACCESS_KEY
  
  executor:
    cores: 1
    instances: 2
    memory: "1000m"
    env:
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: spark-event-logs
          key: AWS_ACCESS_KEY_ID
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: spark-event-logs
          key: AWS_SECRET_ACCESS_KEY
```

**Replace placeholders:**

```bash
# Get values
BUCKET_NAME=$(oc get cm spark-event-logs -n spark-operator -o jsonpath='{.data.BUCKET_NAME}')
S3_ENDPOINT=$(oc get cm spark-event-logs -n spark-operator -o jsonpath='{.data.BUCKET_HOST}')
S3_PORT=$(oc get cm spark-event-logs -n spark-operator -o jsonpath='{.data.BUCKET_PORT}')

# Update the YAML (or use sed/envsubst)
echo "Update spark.eventLog.dir to: s3a://$BUCKET_NAME/spark-event-logs"
echo "Update spark.hadoop.fs.s3a.endpoint to: $S3_ENDPOINT:$S3_PORT"
```

Apply:

```bash
oc apply -f spark-pi-with-odf.yaml
```

### Step 5: Deploy Spark History Server

Create `spark-history-server-odf.yaml`:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spark-history-server
  namespace: spark-operator
  labels:
    app: spark-history-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spark-history-server
  template:
    metadata:
      labels:
        app: spark-history-server
    spec:
      containers:
      - name: spark-history-server
        image: quay.io/opendatahub/spark:3.5.7
        command: ["/bin/bash", "-c"]
        args:
        - |
          /opt/spark/sbin/start-history-server.sh
          tail -f /opt/spark/logs/spark--org.apache.spark.deploy.history.HistoryServer-*.out
        
        env:
        - name: SPARK_NO_DAEMONIZE
          value: "true"
        
        - name: SPARK_HISTORY_OPTS
          value: >-
            -Dspark.history.fs.logDirectory=s3a://BUCKET_NAME/spark-event-logs
            -Dspark.history.ui.port=18080
            -Dspark.hadoop.fs.s3a.endpoint=S3_ENDPOINT:S3_PORT
            -Dspark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem
            -Dspark.hadoop.fs.s3a.path.style.access=true
            -Dspark.hadoop.fs.s3a.connection.ssl.enabled=true
            -Dspark.hadoop.fs.s3a.access.key=${AWS_ACCESS_KEY_ID}
            -Dspark.hadoop.fs.s3a.secret.key=${AWS_SECRET_ACCESS_KEY}
        
        envFrom:
        - secretRef:
            name: spark-event-logs
        
        ports:
        - containerPort: 18080
          name: http
          protocol: TCP
        
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        
        livenessProbe:
          httpGet:
            path: /
            port: 18080
          initialDelaySeconds: 30
          periodSeconds: 10
        
        readinessProbe:
          httpGet:
            path: /
            port: 18080
          initialDelaySeconds: 20
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: spark-history-server
  namespace: spark-operator
  labels:
    app: spark-history-server
spec:
  type: ClusterIP
  selector:
    app: spark-history-server
  ports:
  - port: 18080
    targetPort: 18080
    name: http
    protocol: TCP

---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: spark-history-server
  namespace: spark-operator
  labels:
    app: spark-history-server
spec:
  port:
    targetPort: http
  to:
    kind: Service
    name: spark-history-server
    weight: 100
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
  wildcardPolicy: None
```

**Replace placeholders** same as Step 4, then apply:

```bash
oc apply -f spark-history-server-odf.yaml
```

### Step 6: Verify and Access

```bash
# Check History Server pod
oc get pods -n spark-operator -l app=spark-history-server

# Check logs
oc logs -n spark-operator -l app=spark-history-server --tail=50

# You should see:
# "Bound HistoryServer to 0.0.0.0, and started at http://..."
# "Started HistoryServer"

# Get Route URL
HISTORY_URL="https://$(oc get route spark-history-server -n spark-operator -o jsonpath='{.spec.host}')"
echo "Access History Server at: $HISTORY_URL"
```

Open `$HISTORY_URL` in your browser to view completed Spark applications.

---

## Approach 2: ODF RWX PVC Storage

This approach uses ODF's CephFS to provide a ReadWriteMany PVC.

### Step 1: Create PVC with ODF CephFS

Create `odf-pvc.yaml`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: spark-event-logs
  namespace: spark-operator
spec:
  accessModes:
    - ReadWriteMany  # Required for multiple apps + history server
  resources:
    requests:
      storage: 50Gi
  storageClassName: ocs-storagecluster-cephfs  # ODF CephFS storage class
```

Apply:

```bash
oc apply -f odf-pvc.yaml

# Verify PVC is bound
oc get pvc spark-event-logs -n spark-operator
# STATUS should be "Bound"
```

### Step 2: Configure SparkApplication

Create `spark-pi-with-odf-pvc.yaml`:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi-odf-pvc
  namespace: spark-operator
spec:
  type: Scala
  mode: cluster
  image: quay.io/opendatahub/spark:3.5.7
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples_2.12-3.5.7.jar
  sparkVersion: "3.5.7"
  
  restartPolicy:
    type: Never
  
  sparkConf:
    "spark.eventLog.enabled": "true"
    "spark.eventLog.dir": "file:///mnt/spark-event-logs"
    "spark.eventLog.compress": "true"
  
  driver:
    cores: 1
    memory: "1000m"
    serviceAccount: spark-operator-spark
    volumeMounts:
    - name: event-logs
      mountPath: /mnt/spark-event-logs
  
  executor:
    cores: 1
    instances: 2
    memory: "1000m"
    volumeMounts:
    - name: event-logs
      mountPath: /mnt/spark-event-logs
  
  volumes:
  - name: event-logs
    persistentVolumeClaim:
      claimName: spark-event-logs
```

Apply:

```bash
oc apply -f spark-pi-with-odf-pvc.yaml
```

### Step 3: Deploy History Server with PVC

Create `spark-history-server-odf-pvc.yaml`:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spark-history-server
  namespace: spark-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spark-history-server
  template:
    metadata:
      labels:
        app: spark-history-server
    spec:
      containers:
      - name: spark-history-server
        image: quay.io/opendatahub/spark:3.5.7
        command: ["/bin/bash", "-c"]
        args:
        - |
          /opt/spark/sbin/start-history-server.sh
          tail -f /opt/spark/logs/spark--org.apache.spark.deploy.history.HistoryServer-*.out
        
        env:
        - name: SPARK_NO_DAEMONIZE
          value: "true"
        - name: SPARK_HISTORY_OPTS
          value: >-
            -Dspark.history.fs.logDirectory=file:///mnt/spark-event-logs
            -Dspark.history.ui.port=18080
        
        volumeMounts:
        - name: event-logs
          mountPath: /mnt/spark-event-logs
        
        ports:
        - containerPort: 18080
          name: http
        
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
      
      volumes:
      - name: event-logs
        persistentVolumeClaim:
          claimName: spark-event-logs

---
apiVersion: v1
kind: Service
metadata:
  name: spark-history-server
  namespace: spark-operator
spec:
  selector:
    app: spark-history-server
  ports:
  - port: 18080
    targetPort: 18080
    name: http

---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: spark-history-server
  namespace: spark-operator
spec:
  port:
    targetPort: http
  to:
    kind: Service
    name: spark-history-server
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
```

Apply:

```bash
oc apply -f spark-history-server-odf-pvc.yaml
```

---

## Troubleshooting

### ODF ObjectBucketClaim Stuck in Pending

**Symptom:**
```bash
oc get objectbucketclaim spark-event-logs
# PHASE shows "Pending" instead of "Bound"
```

**Possible causes:**

1. **NooBaa not installed or not healthy**

```bash
# Check NooBaa status
oc get noobaa -n openshift-storage

# Should show: NAME     MGMT-ENDPOINTS              S3-ENDPOINTS                IMAGE    PHASE   AGE
#              noobaa   [https://noobaa-mgmt...]   [https://s3.openshift...]   X.X.X    Ready   XXd
```

If NooBaa is not Ready, check ODF operator logs:

```bash
oc logs -n openshift-storage deployment/odf-operator-controller-manager
```

2. **StorageClass not available**

```bash
# Verify ODF storage classes exist
oc get storageclass | grep openshift-storage

# Should show:
# openshift-storage.noobaa.io
# ocs-storagecluster-ceph-rbd
# ocs-storagecluster-cephfs
```

### Cannot Access S3 Endpoint from Pods

**Symptom:** SparkApplication or History Server fails with:

```
java.net.UnknownHostException: s3.openshift-storage.svc
```

**Solution:** Verify NooBaa service exists:

```bash
oc get svc -n openshift-storage | grep s3

# Should show service "s3" in openshift-storage namespace
```

Full endpoint should be: `s3.openshift-storage.svc:443` (or `:80` for HTTP)

### 403 Access Denied from ODF S3

**Symptom:**
```
Status Code: 403, AWS Service: Amazon S3
```

**Check credentials:**

```bash
# Verify secret has correct keys
oc get secret spark-event-logs -n spark-operator -o yaml

# Should contain:
# AWS_ACCESS_KEY_ID: <base64>
# AWS_SECRET_ACCESS_KEY: <base64>

# Decode to verify
oc get secret spark-event-logs -n spark-operator \
  -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d
```

### PVC Stuck in Pending

**Symptom:**
```bash
oc get pvc spark-event-logs
# STATUS shows "Pending"
```

**Check:**

1. **ODF CephFS storage class exists:**

```bash
oc get storageclass ocs-storagecluster-cephfs
```

2. **ODF storage cluster is healthy:**

```bash
oc get storagecluster -n openshift-storage
# PHASE should be "Ready"
```

3. **Check PVC events:**

```bash
oc describe pvc spark-event-logs -n spark-operator
```

### History Server Shows No Applications

**Verify event logs exist in ODF:**

**For S3 approach:**

```bash
# Get NooBaa CLI pod
NOOBAA_POD=$(oc get pods -n openshift-storage -l app=noobaa-core -o name | head -1)

# List bucket contents
oc exec -n openshift-storage $NOOBAA_POD -- noobaa-cli bucket list

# Or use AWS CLI with ODF credentials
aws s3 ls s3://$BUCKET_NAME/spark-event-logs/ \
  --endpoint-url https://$S3_ENDPOINT \
  --region us-east-1
```

**For PVC approach:**

```bash
# Exec into History Server pod
oc exec -n spark-operator deployment/spark-history-server -- \
  ls -lah /mnt/spark-event-logs/

# Should show .lz4 or .inprogress files
```

---

## Advanced Configuration

### High Availability Deployment

Run multiple History Server replicas:

```yaml
spec:
  replicas: 2  # Multiple replicas for HA
  strategy:
    type: RollingUpdate
```

All replicas read the same ODF storage, so scaling improves availability.

### Event Log Cleanup

Enable automatic cleanup of old logs:

```yaml
env:
- name: SPARK_HISTORY_OPTS
  value: >-
    -Dspark.history.fs.logDirectory=s3a://bucket/logs
    -Dspark.history.fs.cleaner.enabled=true
    -Dspark.history.fs.cleaner.interval=1d
    -Dspark.history.fs.cleaner.maxAge=30d
```

### ObjectBucketClaim with Custom Retention

Create ObjectBucketClaim with lifecycle policies:

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name: spark-event-logs
  namespace: spark-operator
  annotations:
    objectbucket.io/bucket-retention-days: "90"  # Auto-delete after 90 days
spec:
  generateBucketName: spark-logs
  storageClassName: openshift-storage.noobaa.io
```

### Multi-Namespace Setup

For shared History Server across namespaces:

1. Create ObjectBucketClaim in central namespace
2. Copy credentials secret to each application namespace:

```bash
# Copy secret to app namespace
oc get secret spark-event-logs -n spark-operator -o yaml | \
  sed 's/namespace: spark-operator/namespace: team-a/' | \
  oc apply -f -
```

3. Each namespace uses same bucket with different prefix:

```yaml
sparkConf:
  "spark.eventLog.dir": "s3a://shared-bucket/team-a/event-logs"
```

---

## Performance Tuning

### S3 Performance Settings

```yaml
sparkConf:
  "spark.hadoop.fs.s3a.fast.upload": "true"
  "spark.hadoop.fs.s3a.fast.upload.buffer": "disk"
  "spark.hadoop.fs.s3a.threads.max": "20"
  "spark.hadoop.fs.s3a.connection.maximum": "100"
```

### CephFS Mount Options

For PVC approach, tune CephFS mount:

```yaml
volumeMounts:
- name: event-logs
  mountPath: /mnt/spark-event-logs
  mountPropagation: HostToContainer
```

---

## Related Resources

- [ODF Documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/)
- [NooBaa ObjectBucketClaims](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/4.15/html/managing_hybrid_and_multicloud_resources/object-bucket-claim)
- [Spark History Server Configuration](https://spark.apache.org/docs/latest/monitoring.html#viewing-after-the-fact)
- [Back to Overview](../../docs/spark-observability-overview.md)

---

**Document Version:** 1.0  
**Last Updated:** July 6, 2026  
**Tested With:** ODF 4.15, Spark 3.5.7
