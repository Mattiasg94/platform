variable "project_id" {
  description = "GCP project holding the platform."
  type        = string
  default     = "ai-agent-502309"
}

variable "region" {
  description = "Region for every resource. Cloud Run's always-free tier only applies in us-central1, us-east1 and us-west1, and the registry is co-located with the compute so image pulls stay in-region."
  type        = string
  default     = "us-central1"
}

variable "github_repo" {
  description = "owner/name of the GitHub repo allowed to impersonate the CD deployer via workload identity federation."
  type        = string
  default     = "Mattiasg94/platform"
}

variable "project_repo_url" {
  description = "The project repo the orchestrator drives: cloned per run, and its name selects the prebuilt agent-<name> image. Currently the demo project."
  type        = string
  default     = "https://github.com/Mattiasg94/demo-project"
}
