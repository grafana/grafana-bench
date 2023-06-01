provider "google" {
  credentials = file("../GCP-infra-manager-828bbfa6f427.json")
  project = "grafana-bench"
  region  = "us-central1"
  zone    = "us-central1-c"
}

locals {
  service_account = "serviceAccount:infra-manager@grafana-bench.iam.gserviceaccount.com"
}

// create bucket to store builds
resource "google_storage_bucket" "bench_builds" {
  name          = "bench-builds"
  location      = "US"
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
  role = "roles/storage.admin"
  members = [
    local.service_account,
  ]
}

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





// hardcoded for now, this may change in the future when we end up in deployment
// tools
#project_id = "grafana-bench"

// create user to upload builds
#resource "google_service_account" "build_runner" {
#  account_id   = "build-runner"
#  display_name = "build-runner"
#  description = "generic user for grafana-bench to upload and download builds"
#}

#resource "google_service_account_key" "build_runner_key" {
#  service_account_id = google_service_account.build_runner.id
#  key_algorithm      = "RSA"
#}

#output "example_user_key_path" {
#  value = google_service_account_key.build_runner.private_key_file
#}

// Assign runner as manager
#resource "google_storage_bucket_iam_binding" "build_storage_contributor" {
#  bucket = google_storage_bucket.bench_builds.name
#  role = "roles/storage.manager"
#  members = [
#    "user:${google_service_account.build_runner.email}",
#  ]
#}


  #iam_configuration {
    #// read permissions for everyone
    #bindings {
      #role = "roles/storage.objectViewer"
      #members = [
        #"allUsers"
      #]
    #}

    #// write only for admin
    #// TODO lock this down
    #bindings {
      #role = "roles/storage.objectAdmin"
      #members = [
        #"infra-manager@grafana-bench.iam.gserviceaccount.com",
        #"user:${google_service_account.build-runner.email}",
      #]
    #}

    #// give build runner permissions
    #bindings {
      #role = "roles/storage.objectManager"
      #members = ["user:${google_service_account.build-runner.email}"]
    #}
  #}
#}
