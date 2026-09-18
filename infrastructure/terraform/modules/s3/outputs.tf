output "bucket_name" {
  value = aws_s3_bucket.snapshots.bucket
}

output "bucket_arn" {
  value = aws_s3_bucket.snapshots.arn
}
