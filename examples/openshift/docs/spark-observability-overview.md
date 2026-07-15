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

History Server requires persistent storage for event logs. Choose based on your infrastructure:

#### S3-Compatible Object Storage

**What it is:** Any S3-compatible object storage endpoint (AWS S3, OpenShift Data Foundation, cloud providers, enterprise storage vendors)

**Advantages:**
- ✅ Concurrent access from multiple namespaces and applications
- ✅ Unlimited storage scalability
- ✅ Works with any S3-compatible provider
- ✅ Industry-standard protocol

**Considerations:**
- Different S3 providers have different performance, cost, and availability characteristics
- Requires network connectivity to S3 endpoint (use providers that support your connectivity requirements)
- Credentials management required

**Documentation:** [S3 Setup Guide](../spark-history-server/s3/)

**Examples of S3-compatible storage:**
- AWS S3 (cloud)
- OpenShift Data Foundation / Ceph RGW (on-cluster)
- Enterprise storage vendors (NetApp, Pure Storage, Dell EMC)
- Cloud provider object storage (Google Cloud Storage, Azure Blob with S3 compatibility)

---

#### ReadWriteMany (RWX) PVC

**What it is:** Kubernetes Persistent Volume Claim with ReadWriteMany access mode

**Advantages:**
- ✅ Simple setup - no credentials or endpoints to configure
- ✅ Works with any RWX-capable storage provider
- ✅ Native Kubernetes resource

**Considerations:**
- Requires storage provider that supports ReadWriteMany access mode
- Not all Kubernetes storage classes support RWX (check your storage provider)
- Typically namespace-scoped (see Multi-Namespace Considerations below)

**Documentation:** [PVC Setup Guide](../spark-history-server/pvc/)

**Examples of RWX-capable storage:**
- OpenShift Data Foundation / CephFS
- NFS
- Enterprise storage vendors (NetApp, Portworx, IBM Storage Fusion)
- Cloud file storage (AWS EFS, Azure Files, Google Filestore)

---

### Multi-Namespace Considerations

**S3 approach:** Multiple namespaces can write to the same S3 bucket. A single History Server can read logs from applications across all namespaces.

**PVC approach:** PVCs are namespace-scoped. For multi-namespace deployments, you'll need either:
- Separate History Server per namespace, or
- Namespace-shared storage configuration (consult your storage provider documentation)

---

### Disconnected / Air-Gapped Environments

Both the Spark Operator and History Server work in disconnected/air-gapped environments:
- **Neither component requires internet connectivity** by default
- S3 approach: Use on-cluster storage (e.g., OpenShift Data Foundation) or enterprise appliances
- PVC approach: Works with any local RWX storage provider

The only external dependency is container image registry access (can be mirrored to disconnected registry).

---

## Documentation Inventory

### Live Monitoring
- ✅ **[Route Access](../spark-ui/route/)** - Production HTTPS access
- ✅ **[Port-Forward Access](../spark-ui/port-forward/)** - Development/testing

### Post-Mortem (History Server)
- ✅ **[S3 Setup](../spark-history-server/s3/)** - S3-compatible object storage
- ✅ **[PVC Setup](../spark-history-server/pvc/)** - ReadWriteMany PVC storage

### Pending
- 📝 Blog post: "How to Debug Spark Jobs on OpenShift" (observability-focused)

---

## Open Questions

### 1. Storage Backend Strategy ✅ RESOLVED

**Decision:** Document two generic approaches without vendor-specific recommendations.

**Rationale (from product team feedback - July 9, 2026):**
- Focus on S3-compatible storage and RWX PVC as architectural patterns
- No vendor-specific recommendations - architectural decision is customer-specific
- Works with S3-compatible solutions: AWS S3, ODF, enterprise storage vendors, cloud providers
- Works with RWX PVC: ODF/CephFS, NFS, enterprise storage, cloud file storage
- Maintain same documentation stance as OpenShift AI data connections (S3-compatible, not vendor-specific)

**Quote from field team:**
> "We work with S3 compatible solutions. We don't say we work with Portworx, NetApp or whatever... That has been the stance for the past 4-5 years on this matter."

### 2. Disconnected/Air-Gapped Requirements ✅ RESOLVED

**Decision:** Explicitly document that both operator and History Server work in disconnected environments.

**Documentation added:**
> "Both the Spark Operator and History Server work in disconnected/air-gapped environments - neither component requires internet connectivity by default."

**Field feedback:**
> "It's still nice to have a line that says, 'By the way, neither the operator nor the history server by default are calling outside.' So, this totally works in an airgapped environment."

### 3. ROSA Storage Limitations

**Question:** Should we document AWS EFS setup for ROSA customers who want PVC approach?

**Context:**
- ROSA default storage (EBS) is ReadWriteOnce (RWO) - doesn't work for concurrent access
- AWS EFS provides ReadWriteMany (RWX) but requires additional setup
- Current development cluster only has EBS storage

**Decision needed:**
- [ ] Invest time in EFS setup documentation?
- [ ] Or recommend S3 for all ROSA deployments?
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
- ✅ S3: Tested on ROSA cluster with AWS S3
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
