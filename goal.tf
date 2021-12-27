resource "aws_api_gateway_resource" "goals" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id = aws_api_gateway_resource.xeffect.id
  path_part = "goal"
}

resource "aws_api_gateway_resource" "goal" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id = aws_api_gateway_resource.goals.id
  path_part = "{goalId}"
}
