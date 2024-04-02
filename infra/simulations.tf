resource "google_storage_bucket" "bench_simulations" {
  name     = "bench-simulations"
  location = local.location

  force_destroy = false
  lifecycle_rule {
    condition {
      age = 1
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }
}

resource "google_service_account" "simulation_service_account" {
  account_id   = "simulation-service"
  display_name = "bench simulation service"
  project      = local.project_id
}

resource "google_storage_bucket_iam_member" "member" {
  bucket = google_storage_bucket.bench_simulations.name
  role   = "roles/storage.admin"
  member = "serviceAccount:${google_service_account.simulation_service_account.email}"
}

resource "google_service_account_key" "simulation_service_account_key" {
  service_account_id = google_service_account.simulation_service_account.name
}

resource "local_sensitive_file" "simulation_service_account_key_file" {
  content = base64decode(google_service_account_key.simulation_service_account_key.private_key)
  filename          = "${path.module}/simulation_service_account_key.json"
}
