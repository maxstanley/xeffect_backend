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

resource "aws_api_gateway_resource" "goal" {
  path_part = "goal"
  parent_id = aws_api_gateway_resource.xeffect.id

  rest_api_id = aws_api_gateway_rest_api.api.id
}

resource "aws_api_gateway_method" "goal_create" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.goal.id

  http_method = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "goal_create" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.goal.id

  http_method = aws_api_gateway_method.goal_create.http_method
  integration_http_method = "POST"
  type = "AWS_PROXY"
  uri = aws_lambda_function.goal_create.invoke_arn
}
