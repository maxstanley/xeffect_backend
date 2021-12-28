data "archive_file" "goal_action" {
  type = "zip"
  source_file = "goal_action/goal_action"
  output_path = "goal_action/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "goal_action" {
  function_name = "goal_action"
  filename = data.archive_file.goal_action.output_path
  handler = "goal_action"
  source_code_hash = data.archive_file.goal_action.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}
