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

# What the agent actually authenticates with today. It is a subscription token,
# not a service credential — a deliberate prototype trade, and the wrong shape for
# production: it belongs to a person rather than to the service, and it expires
# roughly a year after it is minted, silently.
#
# The API key above is kept, unbound, for exactly that reason. Switching back is
# meant to stay one revert away.
#
#   claude setup-token
#   printf '%s' "$CLAUDE_CODE_OAUTH_TOKEN" | \
#     gcloud secrets versions add claude-code-oauth-token --data-file=-
resource "google_secret_manager_secret" "claude_code_oauth_token" {
  secret_id = "claude-code-oauth-token"

  replication {
    auto {}
  }
}
