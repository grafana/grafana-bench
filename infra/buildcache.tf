// create bucket to store builds
resource "google_storage_bucket" "bench_builds" {
  name                        = "bench-builds"
  location                    = local.location
  uniform_bucket_level_access = true

  // don't allow deleting the bucket once we've started putting
  // builds in it
  force_destroy = false

  // delete aborted uploads after 1 day
  lifecycle_rule {
    condition {
      age = 1
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }

}

// Assign self as admin
resource "google_storage_bucket_iam_binding" "build_storage_owner" {
  bucket = google_storage_bucket.bench_builds.name
  role   = "roles/storage.admin"
  members = [
    local.service_account,
  ]
}

