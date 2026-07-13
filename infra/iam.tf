# Read this file more carefully than the others. It is the only one that can
# hurt you: everything here is an answer to "who is allowed to do what".
#
# The agent runs model-generated code over files it did not write. Treat it as
# the least trustworthy component in the system and give it the smallest
# identity that still lets it do its job. Isolation (gVisor, the VM boundary) is
# rented from Cloud Run and protects the *host* from the agent. This file is the
# other boundary — it protects *us* from the agent — and nobody rents it to you.

resource "google_service_account" "agent_job" {
  account_id   = "agent-job"
  display_name = "Agent job"
  description  = "Identity for the agent Cloud Run Job. Deliberately powerless: it may read and write objects in the runs bucket, and read one secret. Nothing else."
}

# Scoped to the bucket, not the project. The agent can touch objects in this one
# bucket. It cannot list other buckets, create infrastructure, execute jobs, or
# read any other secret. That is the entire blast radius of a compromised or
# prompt-injected agent.
#
# The tempting alternative — roles/storage.admin at the project level — also
# works, is what most tutorials do, and is the mistake.
resource "google_storage_bucket_iam_member" "agent_job_runs" {
  bucket = google_storage_bucket.runs.name
  role   = "roles/storage.objectUser"
  member = google_service_account.agent_job.member
}

# Scoped to one secret, not to Secret Manager as a whole — and to the one secret
# the pod actually reads. The unused API key stays in Secret Manager but is
# deliberately left unbound: an identity that can read a credential it never uses
# is an identity larger than its job.
resource "google_secret_manager_secret_iam_member" "agent_job_claude_token" {
  secret_id = google_secret_manager_secret.claude_code_oauth_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.agent_job.member
}
