# APX Agent Execution Plan

**Blueprint for AI-Assisted Implementation**

**Version:** 1.0
**Last Updated:** 2025-11-11
**Status:** Ready for Agent Execution

---

## Purpose

This document is designed for **AI agents** (human + AI pairs or autonomous agents) to implement the APX platform systematically. Each task is:

1. **Self-contained** - Can be executed independently
2. **Testable** - Has clear acceptance criteria
3. **Tracked** - Has progress markers and completion signals
4. **Documented** - Generates artifacts for future reference

---

## How to Use This Plan

### For Human Coordinators

1. Assign tasks to agents (human engineers or AI agents)
2. Agents mark progress by updating the `Status` field
3. Track completion via acceptance criteria checkboxes
4. Review generated artifacts before marking complete

### For AI Agents

Each task follows this format:

```yaml
Task ID: M1-T1-001
Name: Deploy Edge Gateway to Cloud Run
Agent Type: infrastructure
Priority: P0 (Critical) | P1 (High) | P2 (Medium) | P3 (Low)
Dependencies: [M1-T0-001, M1-T0-002]
Estimated Time: 4 hours
Status: NOT_STARTED | IN_PROGRESS | BLOCKED | REVIEW | COMPLETE

Context:
  - What this task achieves
  - Why it matters
  - Where it fits in the architecture

Prerequisites:
  - Files/resources that must exist first
  - Environment setup required

Steps:
  1. Concrete action with command/code
  2. Another step
  3. Verification step

Acceptance Criteria:
  - [ ] Testable outcome 1
  - [ ] Testable outcome 2

Artifacts:
  - file_path: Description of what's created
  - file_path: Another artifact

Rollback:
  - How to undo this task if needed

Next Tasks:
  - [M1-T1-002] - What comes after
```

### Progress Tracking

Agents MUST update these fields:

```yaml
Status: IN_PROGRESS
Started: 2025-11-11T10:00:00Z
Agent: agent-name or human-name
Progress Notes:
  - 2025-11-11T10:00:00Z: Started task
  - 2025-11-11T11:30:00Z: Completed step 3, blocked on X
  - 2025-11-11T14:00:00Z: Unblocked, continuing
Completed: 2025-11-11T15:00:00Z
```

---

## Milestone 0: Foundation (COMPLETE ✅)

**Status:** ✅ COMPLETE
**Duration:** Week 0-1
**Goal:** Repository, schemas, documentation, local dev environment

### Completed Deliverables

- [x] Monorepo structure (47 directories)
- [x] CRD schemas (Product, Route, PolicyBundle)
- [x] Sample configurations (payments-api.yaml)
- [x] Documentation (README, PRINCIPLES, IMPLEMENTATION_PLAN, GAPS_AND_REGRETS)
- [x] Docker Compose stack (local development)
- [x] Makefile (dev commands)
- [x] Edge scaffold (Envoy config, Dockerfile)
- [x] Router scaffold (Go service structure)

---

## Milestone 1: Edge + Router + Async + Observability

**Timeline:** Weeks 1-4 (4 weeks)
**Goal:** Ultra-thin edge → async queue → worker → streaming response with observability
**Team:** 3-4 agents (infrastructure, backend, observability)

---

### Phase M1-T0: Infrastructure Setup (Week 1)

**Prerequisites:** GCP account, billing enabled, project created

---

#### Task M1-T0-001: GCP Project Initialization

```yaml
Task ID: M1-T0-001
Name: Initialize GCP Project and Enable APIs
Agent Type: infrastructure
Priority: P0
Dependencies: []
Estimated Time: 2 hours
Status: NOT_STARTED

Context:
  This task sets up the foundational GCP project infrastructure.
  All subsequent tasks depend on these APIs and configurations.

Prerequisites:
  - GCP account with billing enabled
  - gcloud CLI installed and authenticated
  - Project ID decided (e.g., apx-dev-<random>)

Steps:
  1. Create GCP project:
     ```bash
     export PROJECT_ID="apx-dev-$(openssl rand -hex 4)"
     gcloud projects create $PROJECT_ID --name="APX Development"
     gcloud config set project $PROJECT_ID
     ```

  2. Link billing account:
     ```bash
     gcloud billing accounts list
     export BILLING_ACCOUNT="<account-id>"
     gcloud billing projects link $PROJECT_ID --billing-account=$BILLING_ACCOUNT
     ```

  3. Enable required APIs:
     ```bash
     gcloud services enable \
       compute.googleapis.com \
       run.googleapis.com \
       container.googleapis.com \
       pubsub.googleapis.com \
       firestore.googleapis.com \
       secretmanager.googleapis.com \
       cloudkms.googleapis.com \
       cloudbuild.googleapis.com \
       cloudtrace.googleapis.com \
       monitoring.googleapis.com \
       logging.googleapis.com \
       storage-api.googleapis.com \
       iam.googleapis.com \
       cloudresourcemanager.googleapis.com
     ```

  4. Wait for API enablement (2-3 minutes):
     ```bash
     sleep 180
     gcloud services list --enabled
     ```

  5. Create .env file:
     ```bash
     cd /Users/agentsy/APILEE
     cp .env.example .env
     sed -i '' "s/your-project-id/$PROJECT_ID/g" .env
     echo "GCP_PROJECT_ID=$PROJECT_ID" >> .env
     ```

Acceptance Criteria:
  - [ ] Project created and billing linked
  - [ ] All 15 APIs enabled and verified
  - [ ] .env file contains correct PROJECT_ID
  - [ ] Can run: gcloud projects describe $PROJECT_ID

Artifacts:
  - .env: Updated with GCP_PROJECT_ID
  - infra/terraform/backend.tf: GCS backend configuration (create in next task)

Rollback:
  ```bash
  gcloud projects delete $PROJECT_ID
  ```

Next Tasks:
  - [M1-T0-002] Terraform Backend Setup
```

---

#### Task M1-T0-002: Terraform Backend Setup

```yaml
Task ID: M1-T0-002
Name: Set Up Terraform Backend (GCS)
Agent Type: infrastructure
Priority: P0
Dependencies: [M1-T0-001]
Estimated Time: 1 hour
Status: NOT_STARTED

Context:
  Terraform state must be stored remotely for team collaboration.
  GCS bucket provides versioning, locking, and disaster recovery.

Prerequisites:
  - M1-T0-001 complete
  - Terraform 1.6+ installed

Steps:
  1. Create GCS bucket for Terraform state:
     ```bash
     export PROJECT_ID=$(grep GCP_PROJECT_ID .env | cut -d '=' -f2)
     export BUCKET_NAME="${PROJECT_ID}-terraform-state"

     gsutil mb -p $PROJECT_ID -l us-central1 gs://$BUCKET_NAME
     gsutil versioning set on gs://$BUCKET_NAME
     ```

  2. Create backend configuration:
     ```bash
     cat > infra/terraform/backend.tf <<EOF
     terraform {
       backend "gcs" {
         bucket = "$BUCKET_NAME"
         prefix = "terraform/state"
       }
     }
     EOF
     ```

  3. Create variables file:
     ```bash
     cat > infra/terraform/terraform.tfvars <<EOF
     project_id = "$PROJECT_ID"
     region     = "us-central1"
     environment = "dev"
     EOF
     ```

  4. Initialize Terraform:
     ```bash
     cd infra/terraform
     terraform init
     ```

Acceptance Criteria:
  - [ ] GCS bucket created with versioning enabled
  - [ ] backend.tf exists and references bucket
  - [ ] terraform init succeeds without errors
  - [ ] State file visible in GCS: gsutil ls gs://$BUCKET_NAME/terraform/state/

Artifacts:
  - infra/terraform/backend.tf: Terraform backend config
  - infra/terraform/terraform.tfvars: Environment variables

Rollback:
  ```bash
  gsutil rm -r gs://$BUCKET_NAME
  ```

Next Tasks:
  - [M1-T0-003] Service Accounts and IAM
```

---

#### Task M1-T0-003: Service Accounts and IAM

```yaml
Task ID: M1-T0-003
Name: Create Service Accounts with Least Privilege IAM
Agent Type: infrastructure
Priority: P0
Dependencies: [M1-T0-002]
Estimated Time: 2 hours
Status: NOT_STARTED

Context:
  Each component (edge, router, workers) runs with a dedicated service account.
  Principle of least privilege: grant only necessary permissions.

Prerequisites:
  - M1-T0-001 complete (IAM API enabled)

Steps:
  1. Create Terraform config for service accounts:
     ```bash
     cat > infra/terraform/iam.tf <<'EOF'
     # Edge Gateway Service Account
     resource "google_service_account" "edge" {
       account_id   = "apx-edge"
       display_name = "APX Edge Gateway"
       description  = "Service account for edge gateway (Envoy)"
     }

     resource "google_project_iam_member" "edge_trace_writer" {
       project = var.project_id
       role    = "roles/cloudtrace.agent"
       member  = "serviceAccount:${google_service_account.edge.email}"
     }

     resource "google_project_iam_member" "edge_metric_writer" {
       project = var.project_id
       role    = "roles/monitoring.metricWriter"
       member  = "serviceAccount:${google_service_account.edge.email}"
     }

     # Router Service Account
     resource "google_service_account" "router" {
       account_id   = "apx-router"
       display_name = "APX Router"
       description  = "Service account for router service"
     }

     resource "google_project_iam_member" "router_firestore_user" {
       project = var.project_id
       role    = "roles/datastore.user"
       member  = "serviceAccount:${google_service_account.router.email}"
     }

     resource "google_project_iam_member" "router_pubsub_publisher" {
       project = var.project_id
       role    = "roles/pubsub.publisher"
       member  = "serviceAccount:${google_service_account.router.email}"
     }

     resource "google_project_iam_member" "router_trace_writer" {
       project = var.project_id
       role    = "roles/cloudtrace.agent"
       member  = "serviceAccount:${google_service_account.router.email}"
     }

     resource "google_project_iam_member" "router_metric_writer" {
       project = var.project_id
       role    = "roles/monitoring.metricWriter"
       member  = "serviceAccount:${google_service_account.router.email}"
     }

     # Worker Service Account
     resource "google_service_account" "worker" {
       account_id   = "apx-worker"
       display_name = "APX Worker"
       description  = "Service account for worker pools"
     }

     resource "google_project_iam_member" "worker_pubsub_subscriber" {
       project = var.project_id
       role    = "roles/pubsub.subscriber"
       member  = "serviceAccount:${google_service_account.worker.email}"
     }

     resource "google_project_iam_member" "worker_storage_viewer" {
       project = var.project_id
       role    = "roles/storage.objectViewer"
       member  = "serviceAccount:${google_service_account.worker.email}"
     }

     resource "google_project_iam_member" "worker_trace_writer" {
       project = var.project_id
       role    = "roles/cloudtrace.agent"
       member  = "serviceAccount:${google_service_account.worker.email}"
     }

     resource "google_project_iam_member" "worker_metric_writer" {
       project = var.project_id
       role    = "roles/monitoring.metricWriter"
       member  = "serviceAccount:${google_service_account.worker.email}"
     }

     # Outputs
     output "edge_service_account_email" {
       value = google_service_account.edge.email
     }

     output "router_service_account_email" {
       value = google_service_account.router.email
     }

     output "worker_service_account_email" {
       value = google_service_account.worker.email
     }
     EOF
     ```

  2. Create variables file:
     ```bash
     cat > infra/terraform/variables.tf <<'EOF'
     variable "project_id" {
       description = "GCP Project ID"
       type        = string
     }

     variable "region" {
       description = "GCP Region"
       type        = string
       default     = "us-central1"
     }

     variable "environment" {
       description = "Environment (dev, staging, production)"
       type        = string
       default     = "dev"
     }
     EOF
     ```

  3. Apply Terraform:
     ```bash
     cd infra/terraform
     terraform plan
     terraform apply -auto-approve
     ```

  4. Save service account emails to .env:
     ```bash
     cd ../..
     terraform -chdir=infra/terraform output -raw edge_service_account_email >> .env
     terraform -chdir=infra/terraform output -raw router_service_account_email >> .env
     terraform -chdir=infra/terraform output -raw worker_service_account_email >> .env
     ```

Acceptance Criteria:
  - [ ] 3 service accounts created (edge, router, worker)
  - [ ] Each has minimal required IAM roles
  - [ ] terraform apply succeeds
  - [ ] Service account emails saved to .env
  - [ ] Verify: gcloud iam service-accounts list | grep apx

Artifacts:
  - infra/terraform/iam.tf: Service account definitions
  - infra/terraform/variables.tf: Terraform variables
  - .env: Updated with service account emails

Rollback:
  ```bash
  cd infra/terraform
  terraform destroy -auto-approve
  ```

Next Tasks:
  - [M1-T0-004] VPC and Networking
  - [M1-T1-001] Edge Gateway Deployment (depends on SA)
```

---

#### Task M1-T0-004: VPC and Networking

```yaml
Task ID: M1-T0-004
Name: Create VPC, Subnets, Cloud NAT
Agent Type: infrastructure
Priority: P0
Dependencies: [M1-T0-002]
Estimated Time: 2 hours
Status: NOT_STARTED

Context:
  Custom VPC provides network isolation and control.
  Private Google Access allows Cloud Run to access GCP services.
  Cloud NAT enables outbound internet without public IPs.

Prerequisites:
  - M1-T0-002 complete (Terraform initialized)

Steps:
  1. Create network Terraform config:
     ```bash
     cat > infra/terraform/network.tf <<'EOF'
     # VPC Network
     resource "google_compute_network" "apx_vpc" {
       name                    = "apx-vpc-${var.environment}"
       auto_create_subnetworks = false
       routing_mode            = "REGIONAL"
     }

     # Subnet for Cloud Run and workers
     resource "google_compute_subnetwork" "apx_subnet" {
       name          = "apx-subnet-${var.region}"
       ip_cidr_range = "10.0.0.0/24"
       region        = var.region
       network       = google_compute_network.apx_vpc.id

       private_ip_google_access = true

       log_config {
         aggregation_interval = "INTERVAL_10_MIN"
         flow_sampling        = 0.5
         metadata             = "INCLUDE_ALL_METADATA"
       }
     }

     # Cloud Router for NAT
     resource "google_compute_router" "apx_router" {
       name    = "apx-router-${var.region}"
       region  = var.region
       network = google_compute_network.apx_vpc.id
     }

     # Cloud NAT
     resource "google_compute_router_nat" "apx_nat" {
       name   = "apx-nat-${var.region}"
       router = google_compute_router.apx_router.name
       region = var.region

       nat_ip_allocate_option             = "AUTO_ONLY"
       source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

       log_config {
         enable = true
         filter = "ERRORS_ONLY"
       }
     }

     # Firewall: Allow health checks from GCP
     resource "google_compute_firewall" "allow_health_checks" {
       name    = "apx-allow-health-checks"
       network = google_compute_network.apx_vpc.name

       allow {
         protocol = "tcp"
         ports    = ["8080", "8081"]
       }

       source_ranges = [
         "35.191.0.0/16",    # GCP health check ranges
         "130.211.0.0/22"
       ]

       target_tags = ["apx-edge", "apx-router"]
     }

     # Firewall: Allow internal communication
     resource "google_compute_firewall" "allow_internal" {
       name    = "apx-allow-internal"
       network = google_compute_network.apx_vpc.name

       allow {
         protocol = "tcp"
         ports    = ["0-65535"]
       }

       allow {
         protocol = "udp"
         ports    = ["0-65535"]
       }

       allow {
         protocol = "icmp"
       }

       source_ranges = ["10.0.0.0/24"]
     }

     # Outputs
     output "vpc_network_name" {
       value = google_compute_network.apx_vpc.name
     }

     output "subnet_name" {
       value = google_compute_subnetwork.apx_subnet.name
     }
     EOF
     ```

  2. Apply Terraform:
     ```bash
     cd infra/terraform
     terraform plan
     terraform apply -auto-approve
     ```

  3. Verify network:
     ```bash
     gcloud compute networks describe apx-vpc-dev
     gcloud compute networks subnets describe apx-subnet-us-central1 --region=us-central1
     ```

Acceptance Criteria:
  - [ ] VPC created (apx-vpc-dev)
  - [ ] Subnet created with private Google access
  - [ ] Cloud NAT configured
  - [ ] Firewall rules allow health checks and internal traffic
  - [ ] Verify: gcloud compute networks list | grep apx

Artifacts:
  - infra/terraform/network.tf: Network configuration

Rollback:
  ```bash
  cd infra/terraform
  terraform destroy -target=google_compute_firewall.allow_internal -auto-approve
  terraform destroy -target=google_compute_firewall.allow_health_checks -auto-approve
  terraform destroy -target=google_compute_router_nat.apx_nat -auto-approve
  terraform destroy -target=google_compute_router.apx_router -auto-approve
  terraform destroy -target=google_compute_subnetwork.apx_subnet -auto-approve
  terraform destroy -target=google_compute_network.apx_vpc -auto-approve
  ```

Next Tasks:
  - [M1-T0-005] Firestore Database
  - [M1-T0-006] Pub/Sub Topics
```

---

#### Task M1-T0-005: Firestore Database

```yaml
Task ID: M1-T0-005
Name: Create Firestore Database for Policy Storage
Agent Type: infrastructure
Priority: P0
Dependencies: [M1-T0-001]
Estimated Time: 1 hour
Status: NOT_STARTED

Context:
  Firestore stores compiled policy bundles for low-latency access.
  Native mode (not Datastore mode) for real-time sync.

Prerequisites:
  - M1-T0-001 complete (Firestore API enabled)

Steps:
  1. Create Firestore database:
     ```bash
     export PROJECT_ID=$(grep GCP_PROJECT_ID .env | cut -d '=' -f2)

     gcloud firestore databases create \
       --location=us-central1 \
       --type=firestore-native \
       --project=$PROJECT_ID
     ```

  2. Create Terraform config for collections (metadata):
     ```bash
     cat > infra/terraform/firestore.tf <<'EOF'
     # Firestore is created via gcloud (can't be managed by Terraform for first creation)
     # This file documents the expected collections

     # Collection: policies
     # Documents: {name}@{version}
     # Schema: See router/internal/policy/store.go PolicyBundle struct

     # Example document ID: pb-pay-v1@1.2.0
     # Fields:
     #   - name: string
     #   - version: string
     #   - hash: string (SHA256)
     #   - compat: string (backward|breaking)
     #   - auth_config: map
     #   - authz_rego: string
     #   - quotas: map
     #   - rate_limit: map
     #   - transforms: array
     #   - observability: map
     #   - security: map
     #   - cache: map
     #   - created_at: timestamp
     #   - updated_at: timestamp
     EOF
     ```

  3. Verify database:
     ```bash
     gcloud firestore databases describe --database=(default) --project=$PROJECT_ID
     ```

  4. Create index configuration:
     ```bash
     cat > infra/firestore.indexes.json <<'EOF'
     {
       "indexes": [
         {
           "collectionGroup": "policies",
           "queryScope": "COLLECTION",
           "fields": [
             {"fieldPath": "name", "order": "ASCENDING"},
             {"fieldPath": "version", "order": "DESCENDING"}
           ]
         },
         {
           "collectionGroup": "policies",
           "queryScope": "COLLECTION",
           "fields": [
             {"fieldPath": "created_at", "order": "DESCENDING"}
           ]
         }
       ]
     }
     EOF

     gcloud firestore indexes composite create --file=infra/firestore.indexes.json
     ```

Acceptance Criteria:
  - [ ] Firestore database created in native mode
  - [ ] Database location is us-central1
  - [ ] Indexes configured for policies collection
  - [ ] Verify: gcloud firestore databases list

Artifacts:
  - infra/terraform/firestore.tf: Collection documentation
  - infra/firestore.indexes.json: Index definitions

Rollback:
  Note: Firestore databases cannot be deleted via API. Must be done in console.
  Alternative: Clear all documents if testing.

Next Tasks:
  - [M1-T2-001] Policy Compiler (will write to this database)
```

---

#### Task M1-T0-006: Pub/Sub Topics and Subscriptions

```yaml
Task ID: M1-T0-006
Name: Create Pub/Sub Topics for Async Queue
Agent Type: infrastructure
Priority: P0
Dependencies: [M1-T0-003]
Estimated Time: 1.5 hours
Status: NOT_STARTED

Context:
  Pub/Sub decouples router from workers.
  Per-region topics for data residency.
  Ordering keys for per-tenant FIFO.
  CMEK encryption for security.

Prerequisites:
  - M1-T0-001 complete (Pub/Sub API enabled)
  - M1-T0-003 complete (service accounts exist)

Steps:
  1. Create Cloud KMS key for encryption:
     ```bash
     cat > infra/terraform/kms.tf <<'EOF'
     # KMS Keyring
     resource "google_kms_key_ring" "apx_keyring" {
       name     = "apx-keyring-${var.environment}"
       location = var.region
     }

     # KMS Key for Pub/Sub
     resource "google_kms_crypto_key" "pubsub_key" {
       name            = "apx-pubsub-key"
       key_ring        = google_kms_key_ring.apx_keyring.id
       rotation_period = "7776000s"  # 90 days

       lifecycle {
         prevent_destroy = true
       }
     }

     # Grant Pub/Sub service account access to key
     resource "google_kms_crypto_key_iam_member" "pubsub_encrypter_decrypter" {
       crypto_key_id = google_kms_crypto_key.pubsub_key.id
       role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
       member        = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
     }

     data "google_project" "project" {
       project_id = var.project_id
     }

     output "kms_key_id" {
       value = google_kms_crypto_key.pubsub_key.id
     }
     EOF
     ```

  2. Create Pub/Sub topics:
     ```bash
     cat > infra/terraform/pubsub.tf <<'EOF'
     # Main request topic (US region)
     resource "google_pubsub_topic" "apx_requests_us" {
       name = "apx-requests-us-${var.environment}"

       message_retention_duration = "86400s"  # 24 hours

       kms_key_name = google_kms_crypto_key.pubsub_key.id

       depends_on = [
         google_kms_crypto_key_iam_member.pubsub_encrypter_decrypter
       ]
     }

     # Subscription for workers
     resource "google_pubsub_subscription" "apx_workers_us" {
       name  = "apx-workers-us-${var.environment}"
       topic = google_pubsub_topic.apx_requests_us.name

       ack_deadline_seconds = 600  # 10 minutes (long-running work)

       message_retention_duration = "86400s"

       enable_message_ordering = true

       retry_policy {
         minimum_backoff = "10s"
         maximum_backoff = "600s"
       }

       expiration_policy {
         ttl = ""  # Never expire
       }

       # Dead letter queue
       dead_letter_policy {
         dead_letter_topic     = google_pubsub_topic.apx_dlq.id
         max_delivery_attempts = 5
       }
     }

     # Dead letter queue
     resource "google_pubsub_topic" "apx_dlq" {
       name = "apx-requests-dlq-${var.environment}"

       message_retention_duration = "604800s"  # 7 days

       kms_key_name = google_kms_crypto_key.pubsub_key.id

       depends_on = [
         google_kms_crypto_key_iam_member.pubsub_encrypter_decrypter
       ]
     }

     resource "google_pubsub_subscription" "apx_dlq_sub" {
       name  = "apx-dlq-sub-${var.environment}"
       topic = google_pubsub_topic.apx_dlq.name

       ack_deadline_seconds = 60

       expiration_policy {
         ttl = ""
       }
     }

     # Outputs
     output "pubsub_topic_requests" {
       value = google_pubsub_topic.apx_requests_us.id
     }

     output "pubsub_subscription_workers" {
       value = google_pubsub_subscription.apx_workers_us.id
     }
     EOF
     ```

  3. Apply Terraform:
     ```bash
     cd infra/terraform
     terraform plan
     terraform apply -auto-approve
     ```

  4. Verify topics:
     ```bash
     gcloud pubsub topics list
     gcloud pubsub subscriptions list
     ```

Acceptance Criteria:
  - [ ] KMS key created for encryption
  - [ ] apx-requests-us topic created with CMEK
  - [ ] apx-workers-us subscription with ordering enabled
  - [ ] Dead letter queue configured
  - [ ] Verify: gcloud pubsub topics describe apx-requests-us-dev

Artifacts:
  - infra/terraform/kms.tf: KMS key configuration
  - infra/terraform/pubsub.tf: Pub/Sub topics and subscriptions

Rollback:
  ```bash
  cd infra/terraform
  terraform destroy -target=google_pubsub_subscription.apx_dlq_sub -auto-approve
  terraform destroy -target=google_pubsub_topic.apx_dlq -auto-approve
  terraform destroy -target=google_pubsub_subscription.apx_workers_us -auto-approve
  terraform destroy -target=google_pubsub_topic.apx_requests_us -auto-approve
  terraform destroy -target=google_kms_crypto_key_iam_member.pubsub_encrypter_decrypter -auto-approve
  terraform destroy -target=google_kms_crypto_key.pubsub_key -auto-approve
  terraform destroy -target=google_kms_key_ring.apx_keyring -auto-approve
  ```

Next Tasks:
  - [M1-T1-003] Router Integration (publishes to topic)
  - [M1-T2-001] Worker Pool (subscribes to topic)
```

---

### Phase M1-T1: Edge and Router Deployment (Week 2)

---

#### Task M1-T1-001: Build and Deploy Edge Gateway

```yaml
Task ID: M1-T1-001
Name: Build Edge Docker Image and Deploy to Cloud Run
Agent Type: backend
Priority: P0
Dependencies: [M1-T0-003, M1-T0-004]
Estimated Time: 3 hours
Status: NOT_STARTED

Context:
  Edge gateway is the entry point for all requests.
  Runs Envoy Proxy with JWT authentication, rate limiting, and OTEL tracing.
  Deployed to Cloud Run for auto-scaling and low operational overhead.

Prerequisites:
  - M1-T0-003 complete (service account exists)
  - M1-T0-004 complete (VPC created)
  - Docker installed locally

Steps:
  1. Update Envoy config with correct router endpoint:
     ```bash
     cd /Users/agentsy/APILEE/edge/envoy

     # Get router service URL (will create in next task, for now use placeholder)
     export ROUTER_URL="apx-router.run.app"

     # Envoy config already exists, verify it references router correctly
     grep "apx-router" envoy.yaml
     ```

  2. Build Docker image:
     ```bash
     cd /Users/agentsy/APILEE/edge

     export PROJECT_ID=$(grep GCP_PROJECT_ID ../../.env | cut -d '=' -f2)
     export IMAGE="gcr.io/$PROJECT_ID/apx-edge:v0.1.0"

     docker build -t $IMAGE .
     ```

  3. Push to Container Registry:
     ```bash
     gcloud auth configure-docker
     docker push $IMAGE
     ```

  4. Create Cloud Run Terraform config:
     ```bash
     cat > ../../infra/terraform/cloud_run_edge.tf <<EOF
     resource "google_cloud_run_service" "edge" {
       name     = "apx-edge"
       location = var.region

       template {
         spec {
           service_account_name = google_service_account.edge.email

           containers {
             image = "$IMAGE"

             ports {
               container_port = 8080
             }

             env {
               name  = "ENVOY_LOG_LEVEL"
               value = "info"
             }

             resources {
               limits = {
                 cpu    = "1000m"
                 memory = "512Mi"
               }
             }
           }

           container_concurrency = 80
           timeout_seconds       = 30
         }

         metadata {
           annotations = {
             "autoscaling.knative.dev/minScale" = "1"
             "autoscaling.knative.dev/maxScale" = "100"
             "run.googleapis.com/vpc-access-connector" = google_vpc_access_connector.apx_connector.id
           }
         }
       }

       traffic {
         percent         = 100
         latest_revision = true
       }
     }

     # VPC Access Connector (required for Cloud Run to access VPC)
     resource "google_vpc_access_connector" "apx_connector" {
       name          = "apx-connector"
       region        = var.region
       network       = google_compute_network.apx_vpc.name
       ip_cidr_range = "10.8.0.0/28"
     }

     # Allow unauthenticated access (will be protected by Cloud Armor)
     resource "google_cloud_run_service_iam_member" "edge_noauth" {
       service  = google_cloud_run_service.edge.name
       location = google_cloud_run_service.edge.location
       role     = "roles/run.invoker"
       member   = "allUsers"
     }

     output "edge_url" {
       value = google_cloud_run_service.edge.status[0].url
     }
     EOF
     ```

  5. Apply Terraform:
     ```bash
     cd ../../infra/terraform
     terraform plan
     terraform apply -auto-approve
     ```

  6. Test edge health:
     ```bash
     export EDGE_URL=$(terraform output -raw edge_url)
     curl $EDGE_URL/health
     # Expected: {"status":"ok","service":"apx-edge"}
     ```

Acceptance Criteria:
  - [ ] Docker image built and pushed to GCR
  - [ ] Cloud Run service deployed successfully
  - [ ] Health check returns 200 OK
  - [ ] Service accessible via HTTPS
  - [ ] Min 1 instance, max 100
  - [ ] VPC connector attached

Artifacts:
  - gcr.io/$PROJECT_ID/apx-edge:v0.1.0: Docker image
  - infra/terraform/cloud_run_edge.tf: Cloud Run configuration

Testing:
  ```bash
  # Health check
  curl $EDGE_URL/health

  # Admin interface (should be blocked from public)
  curl $EDGE_URL:9901/stats  # Should fail

  # Send test request (will fail until router is deployed)
  curl -X POST $EDGE_URL/v1/test \
    -H "Content-Type: application/json" \
    -d '{"test": "data"}'
  ```

Rollback:
  ```bash
  cd infra/terraform
  terraform destroy -target=google_cloud_run_service.edge -auto-approve
  terraform destroy -target=google_vpc_access_connector.apx_connector -auto-approve
  ```

Next Tasks:
  - [M1-T1-002] Update Envoy config with router URL
  - [M1-T1-003] Router Deployment
```

---

#### Task M1-T1-002: Finish Router Implementation

```yaml
Task ID: M1-T1-002
Name: Complete Router Service Implementation
Agent Type: backend
Priority: P0
Dependencies: [M1-T0-003, M1-T0-005, M1-T0-006]
Estimated Time: 4 hours
Status: NOT_STARTED

Context:
  Router service is partially scaffolded.
  Need to implement: route matching, Pub/Sub publishing, missing middleware.

Prerequisites:
  - M1-T0-005 complete (Firestore exists)
  - M1-T0-006 complete (Pub/Sub topics exist)
  - Go 1.22+ installed

Steps:
  1. Implement missing middleware (logging, tracing, metrics):
     ```bash
     cd /Users/agentsy/APILEE/router/internal/middleware

     # Create logging middleware
     cat > logging.go <<'EOF'
     package middleware

     import (
       "net/http"
       "time"
       "go.uber.org/zap"
     )

     type responseWriter struct {
       http.ResponseWriter
       statusCode int
       bytes      int
     }

     func (rw *responseWriter) WriteHeader(code int) {
       rw.statusCode = code
       rw.ResponseWriter.WriteHeader(code)
     }

     func (rw *responseWriter) Write(b []byte) (int, error) {
       n, err := rw.ResponseWriter.Write(b)
       rw.bytes += n
       return n, err
     }

     func Logging(logger *zap.Logger) Middleware {
       return func(next http.Handler) http.Handler {
         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           start := time.Now()

           rw := &responseWriter{ResponseWriter: w, statusCode: 200}

           next.ServeHTTP(rw, r)

           duration := time.Since(start)

           logger.Info("request",
             zap.String("method", r.Method),
             zap.String("path", r.URL.Path),
             zap.Int("status", rw.statusCode),
             zap.Duration("duration", duration),
             zap.Int("bytes", rw.bytes),
             zap.String("request_id", GetRequestID(r.Context())),
             zap.String("tenant_id", GetTenantID(r.Context())),
           )
         })
       }
     }
     EOF

     # Create tracing middleware
     cat > tracing.go <<'EOF'
     package middleware

     import (
       "net/http"
       "go.opentelemetry.io/otel"
       "go.opentelemetry.io/otel/attribute"
       "go.opentelemetry.io/otel/trace"
     )

     func Tracing() Middleware {
       tracer := otel.Tracer("apx-router")

       return func(next http.Handler) http.Handler {
         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           ctx, span := tracer.Start(r.Context(), r.URL.Path)
           defer span.End()

           span.SetAttributes(
             attribute.String("http.method", r.Method),
             attribute.String("http.url", r.URL.String()),
             attribute.String("http.host", r.Host),
           )

           // Add tenant info if available
           if tenantID := GetTenantID(ctx); tenantID != "" {
             span.SetAttributes(attribute.String("tenant.id", tenantID))
           }

           next.ServeHTTP(w, r.WithContext(ctx))
         })
       }
     }
     EOF

     # Create metrics middleware
     cat > metrics.go <<'EOF'
     package middleware

     import (
       "net/http"
       "strconv"
       "time"
       "github.com/prometheus/client_golang/prometheus"
       "github.com/prometheus/client_golang/prometheus/promauto"
     )

     var (
       requestDuration = promauto.NewHistogramVec(
         prometheus.HistogramOpts{
           Name: "apx_request_duration_seconds",
           Help: "Request duration in seconds",
           Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
         },
         []string{"method", "path", "status"},
       )

       requestsTotal = promauto.NewCounterVec(
         prometheus.CounterOpts{
           Name: "apx_requests_total",
           Help: "Total number of requests",
         },
         []string{"method", "path", "status"},
       )
     )

     func Metrics() Middleware {
       return func(next http.Handler) http.Handler {
         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           start := time.Now()

           rw := &responseWriter{ResponseWriter: w, statusCode: 200}
           next.ServeHTTP(rw, r)

           duration := time.Since(start).Seconds()
           status := strconv.Itoa(rw.statusCode)

           requestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
           requestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
         })
       }
     }
     EOF

     # Create policy version tagging middleware
     cat > policy_version.go <<'EOF'
     package middleware

     import (
       "context"
       "net/http"
       "go.uber.org/zap"
     )

     type policyStore interface {
       GetLatestVersion(ctx context.Context, product string) (string, error)
     }

     const PolicyVersionKey contextKey = "policy_version"

     func PolicyVersionTag(store policyStore, logger *zap.Logger) Middleware {
       return func(next http.Handler) http.Handler {
         return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           // For now, use a fixed version
           // TODO: Implement dynamic version lookup based on route
           policyVersion := "1.0.0"

           // Add to context
           ctx := context.WithValue(r.Context(), PolicyVersionKey, policyVersion)

           // Add to response headers
           w.Header().Set("X-APX-Policy-Version", policyVersion)

           logger.Debug("policy version tagged",
             zap.String("version", policyVersion),
             zap.String("request_id", GetRequestID(ctx)),
           )

           next.ServeHTTP(w, r.WithContext(ctx))
         })
       }
     }

     func GetPolicyVersion(ctx context.Context) string {
       if version, ok := ctx.Value(PolicyVersionKey).(string); ok {
         return version
       }
       return "unknown"
     }
     EOF
     ```

  2. Implement route matching and Pub/Sub publishing:
     ```bash
     cd ../routes

     cat > matcher.go <<'EOF'
     package routes

     import (
       "context"
       "encoding/json"
       "net/http"
       "time"

       "cloud.google.com/go/pubsub"
       "github.com/apx/router/internal/config"
       "github.com/apx/router/internal/middleware"
       "github.com/google/uuid"
       "go.uber.org/zap"
     )

     type Matcher struct {
       cfg         *config.Config
       logger      *zap.Logger
       pubsubClient *pubsub.Client
       topic       *pubsub.Topic
     }

     func NewMatcher(ctx context.Context, cfg *config.Config, policyStore interface{}, logger *zap.Logger) (*Matcher, error) {
       client, err := pubsub.NewClient(ctx, cfg.PubSubProjectID)
       if err != nil {
         return nil, err
       }

       topic := client.Topic(cfg.PubSubTopic)

       return &Matcher{
         cfg:         cfg,
         logger:      logger,
         pubsubClient: client,
         topic:       topic,
       }, nil
     }

     func (m *Matcher) Handle(w http.ResponseWriter, r *http.Request) {
       ctx := r.Context()

       // Extract context
       requestID := middleware.GetRequestID(ctx)
       tenantID := middleware.GetTenantID(ctx)
       policyVersion := middleware.GetPolicyVersion(ctx)

       // Create request payload
       payload := map[string]interface{}{
         "request_id":     requestID,
         "tenant_id":      tenantID,
         "policy_version": policyVersion,
         "method":         r.Method,
         "path":           r.URL.Path,
         "headers":        r.Header,
         "timestamp":      time.Now().Unix(),
       }

       payloadBytes, err := json.Marshal(payload)
       if err != nil {
         m.logger.Error("failed to marshal payload", zap.Error(err))
         http.Error(w, "Internal Server Error", http.StatusInternalServerError)
         return
       }

       // Publish to Pub/Sub with ordering key (tenant_id for FIFO)
       msg := &pubsub.Message{
         Data: payloadBytes,
         Attributes: map[string]string{
           "request_id":     requestID,
           "tenant_id":      tenantID,
           "policy_version": policyVersion,
         },
         OrderingKey: tenantID,
       }

       result := m.topic.Publish(ctx, msg)

       // Wait for publish to complete
       _, err = result.Get(ctx)
       if err != nil {
         m.logger.Error("failed to publish message", zap.Error(err))
         http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
         return
       }

       // Return 202 Accepted with request ID
       w.Header().Set("Content-Type", "application/json")
       w.WriteHeader(http.StatusAccepted)
       json.NewEncoder(w).Encode(map[string]interface{}{
         "status":     "accepted",
         "request_id": requestID,
         "message":    "Request queued for processing",
       })

       m.logger.Info("request queued",
         zap.String("request_id", requestID),
         zap.String("tenant_id", tenantID),
       )
     }
     EOF
     ```

  3. Implement observability package:
     ```bash
     cd ../../pkg
     mkdir -p observability
     cd observability

     cat > otel.go <<'EOF'
     package observability

     import (
       "context"
       "github.com/apx/router/internal/config"
       "go.opentelemetry.io/otel"
       "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
       "go.opentelemetry.io/otel/sdk/resource"
       "go.opentelemetry.io/otel/sdk/trace"
       semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
       "go.uber.org/zap"
       "google.golang.org/grpc"
       "google.golang.org/grpc/credentials/insecure"
     )

     func Init(ctx context.Context, cfg *config.Config, logger *zap.Logger) (func(), error) {
       // Create OTLP exporter
       opts := []otlptracegrpc.Option{
         otlptracegrpc.WithEndpoint(cfg.OTELEndpoint),
       }

       if cfg.OTELInsecure {
         opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
       }

       exporter, err := otlptracegrpc.New(ctx, opts...)
       if err != nil {
         return nil, err
       }

       // Create resource
       res, err := resource.New(ctx,
         resource.WithAttributes(
           semconv.ServiceName("apx-router"),
           semconv.ServiceVersion("0.1.0"),
           semconv.DeploymentEnvironment(cfg.Environment),
         ),
       )
       if err != nil {
         return nil, err
       }

       // Create trace provider
       tp := trace.NewTracerProvider(
         trace.WithBatcher(exporter),
         trace.WithResource(res),
         trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRate))),
       )

       otel.SetTracerProvider(tp)

       logger.Info("observability initialized",
         zap.String("otel_endpoint", cfg.OTELEndpoint),
         zap.Float64("sample_rate", cfg.SampleRate),
       )

       // Return shutdown function
       return func() {
         if err := tp.Shutdown(ctx); err != nil {
           logger.Error("failed to shutdown tracer", zap.Error(err))
         }
       }, nil
     }
     EOF
     ```

  4. Update go.mod and download dependencies:
     ```bash
     cd /Users/agentsy/APILEE/router
     go mod tidy
     go mod download
     ```

  5. Build and test locally:
     ```bash
     go build -o bin/router cmd/router/main.go

     # Test with local emulators
     export FIRESTORE_EMULATOR_HOST=localhost:8082
     export PUBSUB_EMULATOR_HOST=localhost:8085
     export GCP_PROJECT_ID=$(grep GCP_PROJECT_ID ../.env | cut -d '=' -f2)

     ./bin/router

     # In another terminal:
     curl http://localhost:8081/health
     ```

Acceptance Criteria:
  - [ ] All middleware implemented (logging, tracing, metrics, policy version)
  - [ ] Route matcher publishes to Pub/Sub
  - [ ] Observability (OTEL) initialized
  - [ ] go mod tidy runs without errors
  - [ ] Local build succeeds
  - [ ] Health check returns 200

Artifacts:
  - router/internal/middleware/logging.go
  - router/internal/middleware/tracing.go
  - router/internal/middleware/metrics.go
  - router/internal/middleware/policy_version.go
  - router/internal/routes/matcher.go
  - router/pkg/observability/otel.go

Testing:
  ```bash
  # Health check
  curl http://localhost:8081/health

  # Test request (with emulators running)
  curl -X POST http://localhost:8081/v1/test \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: test-tenant" \
    -d '{"test": "data"}'

  # Should return 202 Accepted with request_id
  ```

Next Tasks:
  - [M1-T1-003] Deploy Router to Cloud Run
```

---

### Checkpoint: Agent Progress Report Template

After completing each task, agents MUST update this status:

```yaml
Task: M1-T1-002
Status: COMPLETE
Completed: 2025-11-11T16:30:00Z
Agent: backend-agent-1
Duration: 3.5 hours

Checklist:
  - [x] All middleware implemented
  - [x] Route matcher working
  - [x] OTEL initialized
  - [x] Tests passing
  - [x] Documentation updated

Files Modified:
  - router/internal/middleware/logging.go (created)
  - router/internal/middleware/tracing.go (created)
  - router/internal/middleware/metrics.go (created)
  - router/internal/middleware/policy_version.go (created)
  - router/internal/routes/matcher.go (created)
  - router/pkg/observability/otel.go (created)
  - router/go.mod (updated)

Issues Encountered:
  - None

Blockers:
  - None

Next Task:
  - M1-T1-003 (Deploy Router to Cloud Run)

Notes:
  - All tests passing locally with emulators
  - Ready for deployment
```

---

## Continuation Pattern for Remaining Tasks

The plan continues with this structure for:

- **M1-T1-003 through M1-T1-006**: Router deployment, integration testing
- **M1-T2-001 through M1-T2-004**: Worker pools, streaming aggregator
- **M1-T3-001 through M1-T3-006**: Observability (OTEL, Prometheus, Grafana, BigQuery)
- **M1-T4-001 through M1-T4-003**: End-to-end testing, load testing, acceptance

Then:
- **M2-T**: Policy compiler, versioning, canary rollouts (16 tasks)
- **M3-T**: Rate limiting, cost controls (12 tasks)
- **M4-T**: Agents, portal, monetization (18 tasks)
- **M5-T**: Multi-region, WebSocket, optimizer (24 tasks)

**Total: ~100 tasks across 6 months**

---

## Agent Coordination Protocol

### Daily Standups (Async)

Each agent posts to shared log:

```yaml
Date: 2025-11-12
Agent: backend-agent-1

Yesterday:
  - Completed: M1-T1-002 (Router implementation)
  - Blocked: None

Today:
  - Planning: M1-T1-003 (Router deployment)
  - ETA: 3 hours

Needs:
  - Review from infrastructure-agent on Terraform config
```

### Blocking Issues

When blocked:

```yaml
Task: M1-T1-003
Status: BLOCKED
Blocker:
  Type: DEPENDENCY
  Description: "Waiting for M1-T0-006 (Pub/Sub topics) to complete"
  Blocked Since: 2025-11-12T10:00:00Z
  Assigned To: infrastructure-agent-1
  Expected Resolution: 2025-11-12T14:00:00Z
```

### Pull Request / Review Flow

When task generates code:

1. Agent creates branch: `task/M1-T1-002-router-implementation`
2. Agent commits with message: `[M1-T1-002] Implement router middleware and routes`
3. Agent creates PR with checklist from acceptance criteria
4. Another agent (or human) reviews
5. On approval: merge, mark task COMPLETE

---

## Success Metrics Dashboard

Track these metrics continuously:

```yaml
Milestone 1 Progress:
  Tasks Completed: 0/25
  Tasks In Progress: 0/25
  Tasks Blocked: 0/25

  Phase T0 (Infrastructure): 0/6 complete
  Phase T1 (Edge & Router): 0/6 complete
  Phase T2 (Workers): 0/4 complete
  Phase T3 (Observability): 0/6 complete
  Phase T4 (Testing): 0/3 complete

Velocity:
  Tasks per Day: 0
  Average Task Duration: 0h
  Blockers Resolved per Day: 0

Quality:
  Tests Passing: 0/0
  Code Coverage: 0%
  Terraform Plans Successful: 0/0
  Deployments Successful: 0/0

SLOs (Target vs Actual):
  Edge p99 Latency: <20ms | N/A
  Request ID Coverage: 100% | N/A
  BigQuery Cost: <$15/day | N/A
```

---

## Emergency Procedures

### Task Rollback

If a task breaks the system:

1. Mark task: `Status: ROLLING_BACK`
2. Execute rollback commands from task definition
3. Mark task: `Status: ROLLED_BACK`
4. Create incident report
5. Fix and retry

### System-Wide Rollback

If entire milestone needs rollback:

```bash
cd infra/terraform
terraform destroy -auto-approve

# Or selective:
terraform destroy -target=google_cloud_run_service.edge
terraform destroy -target=google_cloud_run_service.router
```

---

## Appendix: Complete Task List (Overview)

### Milestone 1 (25 tasks, 4 weeks)

**Phase T0: Infrastructure (6 tasks)**
- M1-T0-001: GCP Project Init ✅ READY
- M1-T0-002: Terraform Backend ✅ READY
- M1-T0-003: Service Accounts ✅ READY
- M1-T0-004: VPC & Networking ✅ READY
- M1-T0-005: Firestore Database ✅ READY
- M1-T0-006: Pub/Sub Topics ✅ READY

**Phase T1: Edge & Router (6 tasks)**
- M1-T1-001: Edge Deployment ✅ READY
- M1-T1-002: Router Implementation ✅ READY
- M1-T1-003: Router Deployment ⏳ PENDING
- M1-T1-004: Edge→Router Integration ⏳ PENDING
- M1-T1-005: Load Balancer Setup ⏳ PENDING
- M1-T1-006: Cloud Armor WAF ⏳ PENDING

**Phase T2: Workers (4 tasks)**
- M1-T2-001: Worker Pool Implementation ⏳ PENDING
- M1-T2-002: Pub/Sub Subscription ⏳ PENDING
- M1-T2-003: Streaming Aggregator ⏳ PENDING
- M1-T2-004: Worker Deployment ⏳ PENDING

**Phase T3: Observability (6 tasks)**
- M1-T3-001: OTEL Collector Setup ⏳ PENDING
- M1-T3-002: Cloud Monitoring Integration ⏳ PENDING
- M1-T3-003: BigQuery Pipeline ⏳ PENDING
- M1-T3-004: Grafana Dashboards ⏳ PENDING
- M1-T3-005: Alert Rules ⏳ PENDING
- M1-T3-006: Cost Budgets ⏳ PENDING

**Phase T4: Testing (3 tasks)**
- M1-T4-001: Integration Tests ⏳ PENDING
- M1-T4-002: Load Tests ⏳ PENDING
- M1-T4-003: Acceptance Tests ⏳ PENDING

### Milestones 2-5 (75 tasks, 20 weeks)

Detailed task breakdowns available in:
- [M2_TASKS.md](./M2_TASKS.md) - Policy compiler, versioning (16 tasks)
- [M3_TASKS.md](./M3_TASKS.md) - Rate limiting, cost controls (12 tasks)
- [M4_TASKS.md](./M4_TASKS.md) - Agents, portal, monetization (18 tasks)
- [M5_TASKS.md](./M5_TASKS.md) - Multi-region, WebSocket, optimizer (24 tasks)

---

**This blueprint is AI-agent optimized. Each task is executable independently with clear inputs, outputs, and success criteria.**

**Agents: Begin with M1-T0-001 and proceed sequentially within each phase. Report progress daily.**

---

**Document Version:** 1.0
**Last Updated:** 2025-11-11
**Next Review:** After M1 completion
**Maintained by:** Platform Architecture Team
