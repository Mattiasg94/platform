# Project number (differs from the id) — needed for the workload-identity
# principal below. Read it rather than hard-code it.
data "google_project" "current" {}

# The orchestrator — an always-on Cloud Run service (epic 0006).
#
# This is deliberately the shape the Temporal worker will need later, so nothing
# here is throwaway: a min-instance service with CPU always allocated is exactly
# how you host a long-lived poller. For now it hosts the HTTP trigger instead.
#
# Its identity is created here; its *grants* (runs bucket, running the agent job,
# Cloud Build) are added as the pipeline is un-stubbed. The first slice only proves
# the service is alive, so it needs none of them yet.
resource "google_service_account" "orchestrator" {
  account_id   = "orchestrator"
  display_name = "Orchestrator service"
  description  = "Identity for the orchestrator Cloud Run service. Grants are added as the run pipeline moves to the cloud."
}

# The service shell. CD (the deploy workflow) sets the image per commit, so
# Terraform owns everything *except* the image tag and ignores changes to it —
# otherwise every deploy would show up as drift and the next apply would revert it.
# The bootstrap image is Google's public hello container, just so the first apply
# has something that runs; the first CD push replaces it.
#
# The service requires an authenticated caller: there is no allUsers invoker
# binding, which is what "authenticated trigger" means in practice.
resource "google_cloud_run_v2_service" "orchestrator" {
  name     = "orchestrator"
  location = var.region

  # Recreatable from code, and safe to let the first apply create before CD runs.
  deletion_protection = false

  template {
    service_account = google_service_account.orchestrator.email

    # Scale to zero when idle. This is an HTTP-triggered service, so it only needs
    # to exist while a request is in flight: it costs nothing between runs and
    # cold-starts in a second or two on the next trigger. There is nothing to turn
    # off by hand — it turns itself off.
    #
    # The always-on shape (min-instances 1, CPU always allocated) returns when this
    # becomes the Temporal worker, which must poll continuously. A poller can't
    # scale to zero; an HTTP trigger can.
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }

    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }

  depends_on = [google_project_service.enabled]
}

# --- CD identity --------------------------------------------------------------
# A dedicated, narrow identity for the deploy workflow. It is *not* terraform-ci:
# building and deploying an app image is a different job from applying
# infrastructure, and it should not borrow that identity's reach.
resource "google_service_account" "orchestrator_deployer" {
  account_id   = "orchestrator-deployer"
  display_name = "Orchestrator CD"
  description  = "Builds and pushes the orchestrator image and deploys new Cloud Run revisions. Used by the deploy GitHub workflow via workload identity federation."
}

# Push images to the registry.
resource "google_artifact_registry_repository_iam_member" "deployer_writer" {
  location   = google_artifact_registry_repository.platform.location
  repository = google_artifact_registry_repository.platform.name
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.orchestrator_deployer.member
}

# Deploy new revisions of the service (scoped to this one service, not the project).
resource "google_cloud_run_v2_service_iam_member" "deployer_developer" {
  location = google_cloud_run_v2_service.orchestrator.location
  name     = google_cloud_run_v2_service.orchestrator.name
  role     = "roles/run.developer"
  member   = google_service_account.orchestrator_deployer.member
}

# Deploy the service *as* the orchestrator's runtime identity. Without actAs on
# that SA, run.developer cannot set the service's service account.
resource "google_service_account_iam_member" "deployer_actas_orchestrator" {
  service_account_id = google_service_account.orchestrator.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.orchestrator_deployer.member
}

# Let the GitHub repo impersonate the deployer through the existing workload
# identity pool — same keyless pattern as terraform-ci, scoped to this one repo.
resource "google_service_account_iam_member" "deployer_wif" {
  service_account_id = google_service_account.orchestrator_deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.current.number}/locations/global/workloadIdentityPools/github/attribute.repository/${var.github_repo}"
}

# --- Orchestrator runtime grants ----------------------------------------------
# What the service is allowed to do once the pipeline is un-stubbed. Until then
# these sit unused — RUN_PIPELINE gates whether the code paths that need them run,
# so it is safe to grant them ahead of the code that uses them. This is a step up
# in reach from the deliberately-powerless agent, which is expected: the
# orchestrator is the supervisor, not the sandbox.

# Read and write run objects on the blackboard: task in, workspace in, result out.
resource "google_storage_bucket_iam_member" "orchestrator_runs" {
  bucket = google_storage_bucket.runs.name
  role   = "roles/storage.objectUser"
  member = google_service_account.orchestrator.member
}

# The orchestrator no longer builds or inspects images — CI builds them ahead of
# time (see agent-builder in iam.tf), and the orchestrator only references them —
# so it needs neither Cloud Build nor registry-read.

# Update and run the agent Cloud Run job — scoped to that one job, not the project.
# The job now exists ahead of time (agent.tf), so the orchestrator only points it at
# the project's image, sets the run's env, and executes it; it never *creates* a job,
# which is the one action that cannot be scoped below the project. That is what lets
# this grant be resource-scoped — and what lets terraform-ci, which cannot edit
# project-level IAM, apply it at all.
resource "google_cloud_run_v2_job_iam_member" "orchestrator_run" {
  location = google_cloud_run_v2_job.agent.location
  name     = google_cloud_run_v2_job.agent.name
  role     = "roles/run.developer"
  member   = google_service_account.orchestrator.member
}

# Running the agent job means acting as the agent's powerless identity, so the
# orchestrator needs actAs on it — the same shape as the deployer acting as the
# orchestrator.
resource "google_service_account_iam_member" "orchestrator_actas_agent" {
  service_account_id = google_service_account.agent_job.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.orchestrator.member
}

# Read the GitHub token to push branches and open PRs.
resource "google_secret_manager_secret_iam_member" "orchestrator_github_token" {
  secret_id = google_secret_manager_secret.github_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.orchestrator.member
}
