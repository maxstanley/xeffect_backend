variable "lambda_zip_file" {
  type = string
  description = "The filename for the zip file to be created and upload to AWS."
  default = "lambda.zip"
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

resource "aws_iam_role_policy" "lambda" {
  name = "lambda_policy"
  role = aws_iam_role.iam_for_lambda.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
         "dynamodb:BatchGetItem",
         "dynamodb:GetItem",
         "dynamodb:Query",
         "dynamodb:Scan",
         "dynamodb:BatchWriteItem",
         "dynamodb:PutItem",
         "dynamodb:UpdateItem"
        ]
        Resource = [
          aws_dynamodb_table.xeffect.arn,
          "${aws_dynamodb_table.xeffect.arn}/*"
        ]
      }
    ]
  })
}

# API Resource.
resource "aws_api_gateway_rest_api" "api" {
  name = "api"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
  disable_execute_api_endpoint = true
}

# Deploy API to domain.
resource "aws_api_gateway_base_path_mapping" "api" {
  api_id = aws_api_gateway_rest_api.api.id
  domain_name = aws_api_gateway_domain_name.api.domain_name
}

# XEffect Resource.
resource "aws_api_gateway_resource" "xeffect" {
  path_part = "xeffect"
  parent_id = aws_api_gateway_rest_api.api.root_resource_id
  rest_api_id = aws_api_gateway_rest_api.api.id
}

# API Deployment.
resource "aws_api_gateway_deployment" "v1" {
  rest_api_id = aws_api_gateway_rest_api.api.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_integration.version,
      aws_api_gateway_integration.goal_create
    ]))
  }
}

# API Stage.
resource "aws_api_gateway_stage" "v1" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  deployment_id = aws_api_gateway_deployment.v1.id

  stage_name = "v1"
}  

resource "aws_lambda_permission" "version" {
  statement_id = "AllowExectionFromAPIGateway"
  action = "lambda:InvokeFunction"
  function_name = aws_lambda_function.version.function_name
  principal = "apigateway.amazonaws.com"

  source_arn = "${aws_api_gateway_rest_api.api.execution_arn}/*/*/*"
}

resource "aws_lambda_permission" "goal_create" {
  statement_id = "AllowExectionFromAPIGateway"
  action = "lambda:InvokeFunction"
  function_name = aws_lambda_function.goal_create.function_name
  principal = "apigateway.amazonaws.com"

  source_arn = "${aws_api_gateway_rest_api.api.execution_arn}/*/*/*"
}
