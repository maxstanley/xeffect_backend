data "archive_file" "version" {
  type = "zip"
  source_file = "version/version"
  output_path = "version/${var.lambda_zip_file}"
}

resource "aws_lambda_function" "version" {
  function_name = "version"
  filename = data.archive_file.version.output_path
  handler = "version"
  source_code_hash = data.archive_file.version.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}

resource "aws_api_gateway_resource" "version" {
  path_part = "version"
  parent_id = aws_api_gateway_resource.xeffect.id

  rest_api_id = aws_api_gateway_rest_api.api.id
}

resource "aws_api_gateway_method" "version" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.version.id

  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "version" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.version.id

  http_method = aws_api_gateway_method.version.http_method
  integration_http_method = "POST"
  type = "AWS_PROXY"
  uri = aws_lambda_function.version.invoke_arn
}
