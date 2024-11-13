data "netbox_ip_addresses" "example" {
  filter {
    name  = "ip_address"
    value = "10.255.255.254/24"
  }
}