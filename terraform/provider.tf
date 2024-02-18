provider "google" {
  project     = "cetakcopilot"
  region      = "asia-southeast1"
  zone        = "asia-southeast1-a"
  credentials = file("gcp_svc_key.json")
}
