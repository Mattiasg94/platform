# Every Google API is off by default, and a disabled one is invisible to
# `terraform plan` — the plan is a prediction from the provider's schema, not a
# rehearsal against the real API. So a missing API only surfaces at apply, when
# something actually calls it. That is exactly how the first apply died halfway
# through, on the service account.
#
# Declaring them makes enablement part of the plan, and makes the project
# reproducible: delete it, run apply, get everything back.

locals {
  services = [
    "artifactregistry.googleapis.com", # image registry
    "cloudbuild.googleapis.com",       # building the per-project agent image (Path B)
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",            # service accounts
    "iamcredentials.googleapis.com", # the token exchange CI authenticates with
    "logging.googleapis.com",        # where a job's stderr lands
    "run.googleapis.com",            # Cloud Run Jobs
    "secretmanager.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com", # the other half of the token exchange
  ]
}

# for_each, not count: each API gets its own address keyed by name, so adding one
# to the list leaves the others untouched. With count, inserting in the middle
# renumbers everything after it and Terraform proposes destroying and recreating
# resources that never changed.
resource "google_project_service" "enabled" {
  for_each = toset(local.services)

  service = each.value

  # Never turn an API off when destroying. Disabling one can break resources
  # outside this configuration, and it is not what anyone means by "remove my
  # bucket".
  disable_on_destroy = false
}
