# The blackboard.
#
# The agent job and the orchestrator run on different machines at different
# times, so they cannot hand a result to each other directly — there is no pipe
# and no shared daemon. Instead the orchestrator writes a run's input here and
# the agent writes its result here, keyed by run ID. Neither has to be alive
# when the other reads.
resource "google_storage_bucket" "runs" {
  name     = "${var.project_id}-runs"
  location = var.region

  # Permissions come from IAM only, never from per-object ACLs. One place to
  # look when asking "who can read this".
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # Run inputs and results are scratch, not archives. Without this they
  # accumulate forever and quietly become a line on the bill.
  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }
}
