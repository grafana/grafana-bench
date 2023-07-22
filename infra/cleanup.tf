// bucket
resource "google_storage_bucket" "infra_bucket" {
  name = "bench-infra"
  location = local.location
}

// archive
data "archive_file" "cleanup_src" {
  type        = "zip"
  source_dir  = "cleanup_func" # Directory where go code is stored
  output_path = "/tmp/cleanup_func.zip"
}

// bucket object
resource "google_storage_bucket_object" "archive" {
  name   = "${data.archive_file.cleanup_src.output_md5}.zip"
  bucket = google_storage_bucket.infra_bucket.name
  source = data.archive_file.cleanup_src.output_path
}

# function
resource "google_cloudfunctions_function" "cleanup_function" {
  available_memory_mb = 128
  entry_point         = "RunClean"
  ingress_settings    = "ALLOW_INTERNAL_ONLY"

  environment_variables = {
    "PROJECT_ID" = local.project_id
    "REGION" = local.region
    "ZONE" = local.zone
  }

  name                  = "cleanup_func"
  runtime               = "go120"
  timeout               = 360 
  trigger_http          = true
  source_archive_bucket = google_storage_bucket.infra_bucket.name
  source_archive_object = "${data.archive_file.cleanup_src.output_md5}.zip"
}

# IAM Configuration. This allows unauthenticated, public access to the function.
# Change this if you require more control here.
resource "google_cloudfunctions_function_iam_member" "invoker" {
  project        = local.project_id
  region         = local.region
  cloud_function = google_cloudfunctions_function.cleanup_function.name

  role   = "roles/cloudfunctions.invoker"
  member = "allUsers"
}

# This is the service account in which the function will act as.
resource "google_service_account" "cleanup_function_service_account" {
  account_id   = "cleanup-sa"
  description  = "service account for cleaning up project resources"
  display_name = "cleanup function service account"
  project      = local.project_id
}

# scheduler
resource "google_cloud_scheduler_job" "cleanup_job" {
  name             = "cleanup-job-schedule"
  description      = "Trigger the ${google_cloudfunctions_function.cleanup_function.name} hourly"
  #schedule         = "* * * * *" # every minute
  schedule         = "1 0 * * *" # daily at 12:01am
  time_zone        = "America/Anchorage"
  attempt_deadline = "320s"

  http_target {
    http_method = "GET"
    uri         = google_cloudfunctions_function.cleanup_function.https_trigger_url

    #oidc_token {
    #  service_account_email = google_service_account.cleanup_function_service_account.email
    #}
  }
}
