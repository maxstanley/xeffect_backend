data "archive_file" "goal_get" {
  type = "zip"
  source_file = "goal_get/goal_get"
  output_path = "goal_get/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "goal_get" {
  function_name = "goal_get"
  filename = data.archive_file.goal_get.output_path
  handler = "goal_get"
  source_code_hash = data.archive_file.goal_get.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}

resource "aws_api_gateway_method" "goal_get" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.goal.id

  http_method = "GET"
  authorization = "NONE"

  request_parameters = {
    "method.request.path.goalId" = true  
  }
}

resource "aws_api_gateway_integration" "goal_get" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.goal.id

  http_method = aws_api_gateway_method.goal_get.http_method
  integration_http_method = "POST"
  type = "AWS_PROXY"
  uri = aws_lambda_function.goal_get.invoke_arn
}
