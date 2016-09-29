variable "env" {}
resource "aws_dynamodb_table" "settings-table" {
    name = "${var.env}-setting"
    read_capacity = 20
    write_capacity = 10
    hash_key = "nodeID"
    range_key = "key"
    attribute {
      name = "nodeID"
      type = "S"
    }
    attribute {
      name = "key"
      type = "S"
    }
}

resource "aws_dynamodb_table" "settings-config-table" {
    name = "${var.env}-setting-config"
    read_capacity = 20
    write_capacity = 10
    hash_key = "key"
    attribute {
      name = "key"
      type = "S"
    }
}