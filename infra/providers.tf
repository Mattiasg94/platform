terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # State lives in GCS, not on the laptop: it is the map from these declarations
  # to the real resources, and GCS gives it durability and locking so a half-
  # finished apply can't corrupt it. The bucket is bootstrap — created by hand,
  # because Terraform cannot store its state in a bucket it has not created yet.
  backend "gcs" {
    bucket = "ai-agent-502309-tfstate"
    prefix = "platform"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
