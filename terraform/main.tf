# resource "google_compute_instance" "vm_instance" {
#   name         = "terraform-instance"
#   machine_type = "e2-micro"

#   boot_disk {
#     initialize_params {
#       image = "debian-cloud/debian-11"
#     }
#   }

#   network_interface {
#     # A default network is created for all GCP projects
#     # network = google_compute_network.vpc_network.self_link
#     access_config {
#     }
#   }
# }

resource "google_compute_network" "vpc_network" {
  name                    = "terraform-network"
  auto_create_subnetworks = "true"
}

# output "CetakCopilot" {
#   value = {
#     # instance_name = google_compute_instance.vm_instance.name
#     # instance_id   = google_compute_instance.vm_instance.id
#     # instance_zone = google_compute_instance.vm_instance.zone
#     # instace_link  = google_compute_instance.vm_instance.self_link
#     gcp_svc_key = var.gcp_svc_key
#   }
# }
