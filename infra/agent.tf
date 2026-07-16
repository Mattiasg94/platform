# The agent — a Cloud Run Job (epic 0005). Terraform owns the job's *existence*;
# the orchestrator owns its *contents*.
#
# This is the same split the orchestrator service uses above. There, Terraform
# creates the service shell and ignores the image so CD can set it. Here, Terraform
# creates the job shell and ignores the whole template, because the orchestrator is
# the sole author of it: on every run it points the job at the right per-project
# image (agent-<project>) and sets the env, secret, and service account
# (internal/pod/cloudrun.go). So Terraform declares a bootstrap template just valid
# enough to create the job, then looks away; the first run overwrites it.
#
# Why the shell has to exist at all: without a job to update-and-run, the
# orchestrator would have to *create* one at runtime — and "create a job" is an
# action that cannot be scoped below the project. Terraform owning the shell is what
# lets the orchestrator's grant (in orchestrator.tf) drop from project-wide
# run.developer to this one job, which in turn is what terraform-ci — which cannot
# edit project-level IAM — is able to apply.
resource "google_cloud_run_v2_job" "agent" {
  name     = "agent"
  location = var.region

  # Recreatable from code; the orchestrator rebuilds the contents on the next run.
  deletion_protection = false

  template {
    template {
      service_account = google_service_account.agent_job.email
      max_retries     = 0 # the agent is not idempotent — a retry re-burns tokens.

      containers {
        # Bootstrap only. The orchestrator replaces this with the project's prebuilt
        # agent image at run time, which is also what re-resolves the mutable :latest
        # tag so each run pulls the freshest build.
        image = "us-docker.pkg.dev/cloudrun/container/hello"
      }
    }
  }

  lifecycle {
    # The orchestrator is the sole author of the template (image, env, secret, SA),
    # so ignore it wholesale — otherwise a run and the next apply would fight over it,
    # exactly the seam the service uses for its image.
    ignore_changes = [template]
  }

  depends_on = [google_project_service.enabled]
}
