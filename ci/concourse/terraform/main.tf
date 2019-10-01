provider "google" {
  project     = "kf-releases"
  region      = "us-central1"
}

provider "google-beta" {
  project     = "kf-releases"
  region      = "us-central1"
}

resource "random_pet" "kf_test" {
}

resource "google_service_account" "kf_test" {
  account_id   = "gke-kf-concourse-${random_pet.kf_test.id}"
  display_name = "Managed by Terraform in Concourse"
}

resource "google_project_iam_binding" "kf_test" {
  role    = "roles/storage.admin"

  members = [
    "serviceAccount:${google_service_account.kf_test.email}",
  ]
}

resource "google_compute_network" "kf_test" {
  name = "kf-test-${random_pet.kf_test.id}"
}

resource "google_container_cluster" "kf_test" {
  provider = "google-beta"
  name     = "kf-test-${random_pet.kf_test.id}"
  location = "us-central1"

  remove_default_node_pool = true
  initial_node_count = 1

  master_auth {
    username = ""
    password = ""

    client_certificate_config {
      issue_client_certificate = false
    }
  }

  ip_allocation_policy {
    use_ip_aliases = true
  }

  addons_config {
    istio_config {
      disabled = false
    }
    cloudrun_config {
      disabled = false
    }
    http_load_balancing {
      disabled = false
    }
  }

  network = "${google_compute_network.kf_test.self_link}"
}

resource "google_container_node_pool" "kf_test" {
  name       = "kf-test-${random_pet.kf_test.id}"
  location   = "us-central1"
  cluster    = "${google_container_cluster.kf_test.name}"
  node_count = 3

  node_config {
    preemptible  = true
    machine_type = "n1-standard-4"

    metadata = {
      disable-legacy-endpoints = "true"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}
