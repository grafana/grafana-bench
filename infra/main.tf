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
