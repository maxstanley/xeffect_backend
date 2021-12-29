variable "domain" {
  type = string
  default = "maxstanley.uk"
}

variable "subdomain" {
  type = string
  default = "api"
}

# Find Zone information for Cloudflare Domain.
data "cloudflare_zones" "maxstanley" {
  filter {
    name = var.domain 
  }
}

locals {
  zone_id = lookup(data.cloudflare_zones.maxstanley.zones[0], "id")
}

# Create DNS Validation request to create certificate.
resource "aws_acm_certificate" "maxstanley" {
  provider = aws.us

  domain_name = var.domain
  subject_alternative_names = [ "*.${var.domain}" ]
  validation_method = "DNS"
}

# Create Cloudflare DNS Record with the DNS validation information provided.
resource "cloudflare_record" "acm_maxstanley" {
  for_each = {
    for dvo in aws_acm_certificate.maxstanley.domain_validation_options : dvo.domain_name => {
      name = dvo.resource_record_name
      type = dvo.resource_record_type
      value = dvo.resource_record_value
    } if dvo.domain_name == var.domain
  }

  zone_id = local.zone_id

  name = each.value.name
  type = each.value.type
  value = each.value.value
  proxied = false
  ttl = 60
}

# Validate the ownership of the domain to create the certificate.
resource "aws_acm_certificate_validation" "maxstanley" {
  provider = aws.us

  certificate_arn = aws_acm_certificate.maxstanley.arn
  validation_record_fqdns = [for record in cloudflare_record.acm_maxstanley : record.hostname]

  timeouts {
    create = "2m"
  }
}

# Create an API Gateway with the desired hostname and appropriate certificate.
resource "aws_api_gateway_domain_name" "api" {
  certificate_arn = aws_acm_certificate_validation.maxstanley.certificate_arn
  domain_name = "${var.subdomain}.${var.domain}"
}

# Create the DNS Record to point the desired hostname to the API Gateway.
resource "cloudflare_record" "maxstanley" {
  zone_id = local.zone_id

  name = var.subdomain
  value = aws_api_gateway_domain_name.api.cloudfront_domain_name
  type = "CNAME"
  proxied = false
}
