resource "aws_s3_bucket" "docker_compose_bucket" {
  bucket = var.bucket_name

  tags = {
    Name        = "Ewood Docker Compose Files"
    Environment = "production"
  }
}

resource "aws_s3_bucket_versioning" "docker_compose_versioning" {
  bucket = aws_s3_bucket.docker_compose_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "docker_compose_encryption" {
  bucket = aws_s3_bucket.docker_compose_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "docker_compose_block" {
  bucket = aws_s3_bucket.docker_compose_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
