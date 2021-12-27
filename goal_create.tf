data "archive_file" "goal_create" {
  type = "zip"
  source_file = "goal_create/goal_create"
  output_path = "goal_create/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "goal_create" {
  function_name = "goal_create"
  filename = data.archive_file.goal_create.output_path
  handler = "goal_create"
  source_code_hash = data.archive_file.goal_create.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}
