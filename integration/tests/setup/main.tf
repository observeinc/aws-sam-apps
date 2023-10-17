resource "random_pet" "run" {
  length = 2
}

resource "aws_s3_bucket" "source" {
  bucket = "${random_pet.run.id}-source"
}

resource "aws_s3_bucket" "destination" {
  bucket = "${random_pet.run.id}-destination"
}

resource "aws_s3_access_point" "source" {
  bucket = aws_s3_bucket.source.id
  name   = random_pet.run.id
}
