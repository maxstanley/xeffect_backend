variable "lambda_zip_file" {
  type = string
  description = "The filename for the zip file to be created and upload to AWS."
  default = "lambda.zip"
}

data "archive_file" "lambda_zip" {
  type = "zip"
  source_file = "version/version"
  output_path = var.lambda_zip_file
}

resource "aws_lambda_function" "version" {
  function_name = "version"
  filename = var.lambda_zip_file
  handler = "version"
  source_code_hash = data.archive_file.lambda_zip.output_base64sha256
  runtime = "go1.x"
  memory_size = 128
  timeout = 10
  role = aws_iam_role.iam_for_lambda.arn
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "iam_for_lambda"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_api_gateway_rest_api" "api" {
  name = "version_api"
}

resource "aws_api_gateway_resource" "resource" {
  path_part = "version"
  parent_id = aws_api_gateway_rest_api.api.root_resource_id
  rest_api_id = aws_api_gateway_rest_api.api.id
}

resource "aws_api_gateway_method" "method" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.resource.id
  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "intergration" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.resource.id
  http_method = aws_api_gateway_method.method.http_method
  integration_http_method = "POST"
  type = "AWS_PROXY"
  uri = aws_lambda_function.version.invoke_arn
}

resource "aws_lambda_permission" "permission" {
  statement_id = "AllowExectionFromAPIGateway"
  action = "lambda:InvokeFunction"
  function_name = aws_lambda_function.version.function_name
  principal = "apigateway.amazonaws.com"

  source_arn = "${aws_api_gateway_rest_api.api.execution_arn}/*/*/*"
}

resource "aws_api_gateway_deployment" "version_deploy" {
  depends_on = [ aws_api_gateway_integration.intergration ]
  rest_api_id = aws_api_gateway_rest_api.api.id
  stage_name = "v1"
}

resource "aws_api_gateway_base_path_mapping" "api" {
  api_id = aws_api_gateway_rest_api.api.id
  domain_name = aws_api_gateway_domain_name.api.domain_name
  base_path = "xeffect"  
}

output "url" {
  value = "${aws_api_gateway_deployment.version_deploy.invoke_url}${aws_api_gateway_resource.resource.path}"
}

