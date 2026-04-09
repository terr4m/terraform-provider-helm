resource "helm_release" "example" {
  chart     = "oci://ghcr.io/fluent/helm-charts/fluent-bit-aggregator"
  name      = "fluent-bit-aggregator"
  namespace = "logging"
  version   = "1.0.2"

  values = {}

  timeouts = {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}
