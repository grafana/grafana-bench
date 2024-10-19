// Create bucket to store VM state
resource "google_compute_network" "vpc_network" {
  name                    = "vm-network"
  auto_create_subnetworks = "true"
}

resource "google_storage_bucket" "bench_vms" {
  name          = "bench-vms"
  location      = "US"
  uniform_bucket_level_access = true

  // don't allow deleting the bucket once we've started putting
  // builds in it
  force_destroy = true

  // delete aborted uploads after 1 day
  lifecycle_rule {
    condition {
      age = 1
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }

  // enable versioning in case state gets deleted
  versioning {
    enabled = true
  }
}

// Assign self as admin
resource "google_storage_bucket_iam_binding" "vm_state_owner" {
  bucket = google_storage_bucket.bench_vms.name
  role = "roles/storage.admin"
  members = [
    local.service_account,
  ]
}
