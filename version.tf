data "archive_file" "version" {
  type = "zip"
  source_file = "version/version"
  output_path = "version/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "xeffect_version" {
  function_name = "version"
  filename = data.archive_file.version.output_path
  handler = "version"
  source_code_hash = data.archive_file.version.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}
