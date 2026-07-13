# Terraform creates the secret *container*, never the secret *value*.
#
# A value written here would land in the Terraform state file in plaintext, and
# the state file would itself become a credential. So the version is added out
# of band, once, by hand:
#
#   printf '%s' "$ANTHROPIC_API_KEY" | \
#     gcloud secrets versions add anthropic-api-key --data-file=-
#
# Rotating the key later means adding a new version — no Terraform run, no
# redeploy.
resource "google_secret_manager_secret" "anthropic_api_key" {
  secret_id = "anthropic-api-key"

  replication {
    auto {}
  }
}
