resource "random_pet" "run" {
  length = 2
}

resource "aws_s3_bucket" "source" {
  bucket        = "${random_pet.run.id}-source"
  force_destroy = true
}

resource "aws_s3_bucket" "destination" {
  bucket        = "${random_pet.run.id}-destination"
  force_destroy = true
}

resource "aws_s3_access_point" "destination" {
  bucket = aws_s3_bucket.destination.id
  name   = random_pet.run.id
}
