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
