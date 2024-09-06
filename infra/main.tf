terraform {
  required_version = "1.5.7"
  required_providers {
    local = {
      source = "hashicorp/local"
        version = "2.5.1"
    }
    archive = {
      source = "hashicorp/archive"
      version = "2.4.2"
    }
    google = {
      source = "hashicorp/google"
      version = "5.33.0"
    }
  }
}

provider "google" {
  credentials = file("../creds/GCP-infra-manager-828bbfa6f427.json")
  project = local.project_id
  region  = local.region
  zone    = local.zone
}

locals {
  service_account = "serviceAccount:infra-manager@grafana-bench.iam.gserviceaccount.com"
  project_id    = "grafana-bench"
  region        = "us-central1"
  zone = "us-central1-c"
  location = "US"
}
