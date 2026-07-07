# Spark Observability on OpenShift - Overview

This document provides a high-level overview of Spark job observability capabilities for OpenShift AI.

## Feature Summary

**Purpose:** Enable data engineers and platform operators to monitor and debug Spark jobs running via the Kubeflow Spark Operator on OpenShift AI.

**Two capabilities:**

1. **Live Monitoring** - Access Spark Application UI while jobs are running
2. **Post-Mortem Analysis** - Use Spark History Server to investigate completed/failed jobs

---

## Live Monitoring - Spark Application UI

Access the built-in Spark UI for running jobs to view:
- Real-time stage progress and task execution
- Executor metrics (CPU, memory, shuffle)
- SQL query plans (for Spark SQL workloads)
- Event timeline and DAG visualization

### Access Methods

| Method | Use Case |
|--------|----------|
| **OpenShift Route** | Production - shareable HTTPS URLs |
| **Port-Forward** | Development/testing - quick local access |

**Recommendation:** Route-based access for production, port-forward for development.

---

## Post-Mortem Analysis - Spark History Server

Deploy a persistent History Server to view completed job details after driver pods terminate.

### Storage Backend Options

History Server requires persistent storage for event logs. **Recommended options:**

#### 1. OpenShift Data Foundation (ODF) - Recommended for OpenShift

**What it is:** Red Hat's Ceph-based storage solution providing both S3-compatible object storage and ReadWriteMany (RWX) PVCs

**Pros:**
- ✅ Fully supported by Red Hat with enterprise SLAs
- ✅ Works in disconnected/air-gapped environments
- ✅ Provides both S3 and file storage from one system
- ✅ No per-GB cloud storage costs
- ✅ Portable across on-prem and cloud deployments

**Cons:**
- ❌ Requires ODF installation and cluster resources

**Best for:** Production OpenShift deployments (on-premises or cloud), disconnected environments

**Documentation:** [ODF Setup Guide](../spark-history-server/odf/)

---

#### 2. AWS S3 (Recommended for ROSA)

**What it is:** AWS-managed object storage

**Pros:**
- ✅ Native to ROSA/AWS environments
- ✅ Unlimited concurrent access
- ✅ Pay-as-you-go ($0.023/GB/month)
- ✅ Scalable, no capacity planning

**Cons:**
- ❌ Requires internet connectivity (not for air-gapped)

**Best for:** ROSA clusters in connected environments

**Documentation:** [S3 Setup Guide](../spark-history-server/s3/)

---

#### 3. Partner Storage Solutions

**What it is:** Enterprise storage from Red Hat partners (NetApp, Portworx, IBM Storage Fusion)

**Pros:**
- ✅ Field-proven in production environments
- ✅ Enterprise support and SLAs
- ✅ Provide both S3 and RWX PVC options

**Best for:** Customers with existing partner storage investments

**Documentation:** Follow partner-specific documentation for S3 or PVC setup, then use corresponding guide above

---

#### 4. MinIO (Self-Hosted Object Storage)

> ⚠️ **Not Recommended for Production:** MinIO is not recommended for production deployments due to licensing changes. For production-grade S3-compatible storage on OpenShift, use **OpenShift Data Foundation (ODF)** instead.

**What it is:** Open-source S3-compatible server running in-cluster

**Pros:**
- ✅ Works in disconnected/air-gapped environments
- ✅ S3-compatible API (same code as AWS S3)
- ✅ Enables concurrent access even on RWO storage

**Cons:**
- ❌ Not recommended for production (licensing concerns)
- ❌ Requires deployment and management of MinIO pods
- ❌ Uses cluster compute resources
- ❌ More complex than ODF or direct PVC

**Best for:** Development/testing only, or legacy environments already using MinIO

**Documentation:** [MinIO Setup Guide](../spark-history-server/minio/)

---

#### 5. PVC (Persistent Volume Claim)

**What it is:** Direct file system storage using Kubernetes PVCs

**Pros:**
- ✅ Simple setup, no credentials
- ✅ Works in disconnected environments
- ✅ No additional infrastructure (with RWX storage)

**Cons:**
- ❌ Requires ReadWriteMany (RWX) storage for production
- ❌ ROSA default storage (EBS) is ReadWriteOnce (RWO) only
- ❌ Limited to cluster storage capacity

**Best for:** Disconnected clusters with NFS or OpenShift Data Foundation (ODF)

**Documentation:** [PVC Setup Guide](../spark-history-server/pvc/)

---

## Storage Decision Tree

```
Running on OpenShift?
│
├─ Yes
│  │
│  ├─ Is ODF (OpenShift Data Foundation) installed?
│  │  │
│  │  ├─ Yes → Use ODF (S3 or PVC mode) ✅ RECOMMENDED
│  │  │
│  │  └─ No
│  │     │
│  │     ├─ Running on ROSA?
│  │     │  └─→ Use AWS S3
│  │     │
│  │     ├─ Have partner storage (NetApp/Portworx/IBM)?
│  │     │  └─→ Use partner S3 or RWX PVC
│  │     │
│  │     └─ Other on-prem/cloud?
│  │        ├─→ Has RWX storage? → Use PVC
│  │        └─→ Consider installing ODF (recommended)
│  │
│  └─ Disconnected/Air-Gapped?
│     │
│     ├─ ODF installed? → Use ODF ✅ BEST CHOICE
│     │
│     └─ No ODF?
│        ├─→ Has RWX storage? → Use PVC
│        └─→ Development only → MinIO (not for production)
│
└─ No (Other Kubernetes)
   └─→ Refer to upstream Spark documentation
```

---

## Environment Support

| Environment | Primary Recommendation | Alternative Options |
|-------------|----------------------|---------------------|
| **OpenShift with ODF** | ODF (S3 or PVC mode) | - |
| **ROSA (Connected)** | AWS S3 | ODF if installed |
| **ROSA (Disconnected)** | ODF | AWS S3 (with VPC endpoints) |
| **On-Prem with Partner Storage** | Partner S3/PVC | ODF |
| **On-Prem with RWX** | ODF (if available), else PVC | - |
| **Disconnected/Air-Gapped** | ODF | PVC with RWX storage |
| **Development/Testing** | Any storage backend | MinIO (not for production) |

---

## Documentation Inventory

### Live Monitoring
- ✅ **[Route Access](../spark-ui/route/)** - Production HTTPS access
- ✅ **[Port-Forward Access](../spark-ui/port-forward/)** - Development/testing

### Post-Mortem (History Server)
- 📝 **[ODF Setup](../spark-history-server/odf/)** - Primary recommendation (in progress)
- ✅ **[S3 Setup](../spark-history-server/s3/)** - For ROSA/connected clusters
- ✅ **[Partner Storage](../spark-history-server/)** - Use ODF/S3/PVC guides with partner-specific setup
- ✅ **[PVC Setup](../spark-history-server/pvc/)** - For clusters with RWX storage
- ⚠️ **[MinIO Setup](../spark-history-server/minio/)** - Development only (not production)

### Pending
- 📝 Blog post showing end-to-end workflow
- 📝 Demo recording/walkthrough

---

## Open Questions

### 1. Storage Backend Strategy ✅ RESOLVED

**Decision:** OpenShift Data Foundation (ODF) is the primary recommended storage solution.

**Rationale (from product feedback):**
- ODF provides enterprise-supported storage with no per-GB costs
- MinIO is not recommended for production due to licensing changes
- Partner solutions (NetApp, Portworx, IBM Fusion) are field-proven alternatives
- AWS S3 remains primary recommendation for ROSA clusters

**Priority:**
1. **Production:** ODF (OpenShift), AWS S3 (ROSA), Partner Storage
2. **Development/Testing:** PVC, MinIO (with warnings)

### 2. Disconnected/Air-Gapped Requirements

**Question:** How important is disconnected cluster support for target customers?

**Context:**
- MinIO and PVC are designed for air-gapped environments
- S3 requires internet connectivity (unless using AWS VPC endpoints)
- Many enterprise/government customers require air-gapped deployments

**Input needed:**
- [ ] What percentage of target customers are air-gapped?
- [ ] Should we create separate documentation tracks for connected vs disconnected?

### 3. ROSA Storage Limitations

**Question:** Should we document AWS EFS setup for ROSA customers who want PVC approach?

**Context:**
- ROSA default storage (EBS) is ReadWriteOnce (RWO) - doesn't work for concurrent access
- AWS EFS provides ReadWriteMany (RWX) but requires additional setup
- Current development cluster only has EBS storage

**Decision needed:**
- [ ] Invest time in EFS setup documentation?
- [ ] Or recommend S3/MinIO for all ROSA deployments?
- [ ] Is there customer demand for PVC-based approach on ROSA?

### 4. IAM Roles for Service Accounts (IRSA)

**Question:** Should we validate and document IRSA for credential-free S3 access?

**Context:**
- Current S3 setup uses AWS access keys stored in Kubernetes secrets
- IRSA allows pods to assume IAM roles via OIDC - eliminates credential secrets
- Draft guide exists but hasn't been tested on ROSA cluster
- More secure approach, better aligned with AWS best practices

**Decision needed:**
- [ ] Worth investing time to test and validate IRSA approach?
- [ ] Should IRSA be recommended over access keys?
- [ ] Is this a customer requirement or nice-to-have?

### 5. Multi-Tenancy Topologies

**Question:** Should we document per-namespace vs centralized History Server deployments?

**Context:**
- Large customers (e.g., RBC) have hundreds of teams/namespaces
- Per-namespace: Simple RBAC but wasteful (100 pods for 100 teams)
- Centralized: Efficient but requires access control (OAuth proxy)

**Scope decision:**
- [ ] Is multi-tenancy in scope for initial release?
- [ ] Or defer to follow-up iteration?
- [ ] Do we need Red Hat-specific UI customizations for access control?

### 6. Production Readiness

**Question:** What level of validation/testing is required before declaring each storage backend "production-ready"?

**Current status:**
- ✅ S3: Tested on ROSA cluster with default EBS storage
- ✅ MinIO: Tested on ROSA cluster with default EBS (RWO) - successfully handles concurrent access
- ⚠️ PVC: Documented, tested on RWO (demo only), not tested on RWX (production scenario)

**Input needed:**
- [ ] What level of validation/testing is required before declaring production-ready?

---

## Success Criteria

From the original RFE:

**Live Spark Application UI:**
- ✅ Documentation published for Route access (production)
- ✅ Documentation published for port-forward access (development)
- ✅ Users can successfully access running job UI

**Spark History Server:**
- ✅ Documentation published for History Server deployment
- ✅ Event log configuration documented
- ✅ Users can successfully view completed/failed job history

**Blog and Demo:**
- 📝 Blog post showing end-to-end workflow (pending)
- 📝 Demo recording or walkthrough (pending)

---

## Next Steps

**Next Steps:**

1. Review storage decision tree - does it align with target customer profiles?
2. Prioritize open questions - which need answers before release?
3. Define support boundaries - what's "supported" vs "community-documented"?
4. Complete blog post and demo
5. Additional testing based on feedback

---

## References

- **RFE:** RHAIRFE-1478
- **Kubeflow Spark Operator Docs:** https://www.kubeflow.org/docs/components/spark-operator/

---

**Last Updated:** June 30, 2026
