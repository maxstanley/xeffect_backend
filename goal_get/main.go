package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Request events.APIGatewayProxyRequest
type Response events.APIGatewayProxyResponse

var GOAL_TABLE = "xeffect_goals"

type Goal struct {
	Title      string `json:"title" validate:"required"`
	Motivation string `json:"motivation" validate:"required"`
}

func returnError(err error) (Response, error) {
	return Response{
		StatusCode:      400,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: err.Error(),
	}, nil
}

func handleGoalGetEvent(ctx context.Context, event Request) (Response, error) {
	goalId := event.PathParameters["goalId"]

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-2"))
	if err != nil {
		return returnError(err)
	}

	input := &dynamodb.GetItemInput{
		TableName: &GOAL_TABLE,
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: goalId,
			},
		},
	}

	client := dynamodb.NewFromConfig(cfg)
	result, err := client.GetItem(ctx, input)
	if err != nil {
		return returnError(err)
	}

	var goal Goal
	if err := attributevalue.UnmarshalMap(result.Item, &goal); err != nil {
		return returnError(err)
	}

	body, err := json.Marshal(goal)
	if err != nil {
		return returnError(err)
	}

	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(handleGoalGetEvent)
}
