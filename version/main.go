package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Request events.APIGatewayProxyRequest
type Response events.APIGatewayProxyResponse

func handleVersionEvent(ctx context.Context, event Request) (Response, error) {
	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: "0.1.0",
	}, nil
}

func main() {
	lambda.Start(handleVersionEvent)
}
