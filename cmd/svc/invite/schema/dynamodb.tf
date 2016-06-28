variable "env" {}
resource "aws_dynamodb_table" "invite-table" {
    name = "${var.env}-invite"
    read_capacity = 1
    write_capacity = 1
    hash_key = "InviteToken"
    attribute {
      name = "InviteToken"
      type = "S"
    }
}

resource "aws_dynamodb_table" "invite-attribution-table" {
    name = "${var.env}-invite-attribution"
    read_capacity = 1
    write_capacity = 1
    hash_key = "DeviceID"
    attribute {
      name = "DeviceID"
      type = "S"
    }
}

resource "aws_dynamodb_table" "entity-token-table" {
    name = "${var.env}-entity-token"
    read_capacity = 1
    write_capacity = 1
    hash_key = "EntityID"
    attribute {
      name = "EntityID"
      type = "S"
    }
}