provider "google" {
  credentials = file("../GCP-infra-manager-828bbfa6f427.json")
  project = "grafana-bench"
  region  = "us-central1"
  zone    = "us-central1-c"
}

provider "tls" {}

# network
resource "google_compute_network" "vpc_network" {
  name                    = "terraform-network"
  auto_create_subnetworks = "true"
}

resource "google_compute_address" "static_ip" {
  name = "bench-vm"
}

resource "google_compute_firewall" "ssh-rule" {
  name = "bench-ssh"
  network = google_compute_network.vpc_network.name
  target_tags = ["bench-vm-instance"]
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "tcp"
    ports = ["22"]
  }

  allow {
    protocol = "icmp"
  }
}

# ssh 
resource "tls_private_key" "ssh_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

data "google_client_openid_userinfo" "me" {}

# vm
resource "google_compute_instance" "vm_instance" {
  name         = "linux-bench"
  machine_type = "e2-micro"

  tags = ["bench-vm-instance"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  metadata = {
    ssh-keys = "${split("@", data.google_client_openid_userinfo.me.email)[0]}:${tls_private_key.ssh_key.public_key_openssh}"
  }

  network_interface {
    network = google_compute_network.vpc_network.name
    access_config {
      nat_ip = google_compute_address.static_ip.address
    }
  }
}

# IP of the VM
output "ssh_connection_string" {
  value = "ssh -i ${split("@", data.google_client_openid_userinfo.me.email)[0]}@${google_compute_address.static_ip.address} -i bench_vm/key"
}

# Output VM config
resource "local_file" "ip_address" {
  content         = google_compute_address.static_ip.address
  filename        = "bench_vm/ip_address"
}

resource "local_file" "ssh_private_key" {
  content         = tls_private_key.ssh_key.private_key_openssh
  filename        = "bench_vm/key"
  file_permission = "0600"
}

resource "local_file" "ssh_public_key" {
  content         = tls_private_key.ssh_key.public_key_openssh
  filename        = "bench_vm/key.pub"
  file_permission = "0600"
}
