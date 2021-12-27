data "archive_file" "goal_get_all" {
  type = "zip"
  source_file = "goal_get_all/goal_get_all"
  output_path = "goal_get_all/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "goal_get_all" {
  function_name = "goal_get_all"
  filename = data.archive_file.goal_get_all.output_path
  handler = "goal_get_all"
  source_code_hash = data.archive_file.goal_get_all.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}
