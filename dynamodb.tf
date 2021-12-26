variable "xeffect_goals_table" {
  type = string
  default = "xeffect_goals"
}

resource "aws_dynamodb_table" "xeffect" {
  name = var.xeffect_goals_table
  hash_key = "uuid"
  billing_mode = "PROVISIONED"
  read_capacity = 1
  write_capacity = 1

  attribute {
    name = "uuid"
    type = "S"
  }
}
