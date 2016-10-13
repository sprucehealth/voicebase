variable "env" {}
resource "aws_dynamodb_table" "invite" {
    name = "${var.env}-invite"
    read_capacity = 10
    write_capacity = 2
    hash_key = "InviteToken"
    attribute {
      name = "InviteToken"
      type = "S"
    }
     attribute {
      name = "ParkedEntityID"
      type = "S"
    }
    global_secondary_index {
      name = "${var.env}-parked_entity_id-index"
      hash_key = "ParkedEntityID"
      write_capacity = 2
      read_capacity = 10
      projection_type = "ALL"
    }
}

resource "aws_dynamodb_table" "attribution" {
    name = "${var.env}-invite-attribution"
    read_capacity = 10
    write_capacity = 2
    hash_key = "DeviceID"
    attribute {
      name = "DeviceID"
      type = "S"
    }
}

resource "aws_dynamodb_table" "entity-token" {
    name = "${var.env}-entity-token"
    read_capacity = 10
    write_capacity = 2
    hash_key = "EntityID"
    attribute {
      name = "EntityID"
      type = "S"
    }
}
