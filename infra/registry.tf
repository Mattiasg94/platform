# Where built images live. Both the agent and the orchestrator push here, and
# Cloud Run pulls from here — the registry is the handoff between "built" and
# "running", so it belongs to neither component.

resource "google_artifact_registry_repository" "platform" {
  repository_id = "platform"
  location      = var.region
  format        = "DOCKER"
  description   = "Container images for the agent platform"

  # Every build pushes layers, and layers are storage that is billed forever
  # unless something removes them. Keep the last 10 versions of each image.
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }

  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"
    condition {
      older_than = "2592000s" # 30 days
    }
  }
}
