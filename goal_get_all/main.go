package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Request events.APIGatewayProxyRequest
type Response events.APIGatewayProxyResponse

var GOAL_TABLE = "xeffect_goals"

type Goal struct {
	Uuid        string                `json:"uuid"`
	Title       string                `json:"title" validate:"required"`
	Motivation  string                `json:"motivation" validate:"required"`
	BestStreak  int                   `json:"best_streak"`
	Streaks     map[string]GoalStreak `json:"streaks"`
	StreakDates []string              `json:"streak_dates"`
}

type GoalStreak struct {
	Length  int               `json:"streak_length"`
	Partial map[string]string `json:"partial"`
}

func returnError(err error) (Response, error) {
	return Response{
		StatusCode:      400,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type":                "text/plain",
			"Access-Control-Allow-Origin": "*",
		},
		Body: err.Error(),
	}, nil
}

func handleGoalGetAllEvent(ctx context.Context, event Request) (Response, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-2"))
	if err != nil {
		return returnError(err)
	}

	input := &dynamodb.ScanInput{
		TableName: &GOAL_TABLE,
	}

	client := dynamodb.NewFromConfig(cfg)
	result, err := client.Scan(ctx, input)
	if err != nil {
		return returnError(err)
	}

	goals := []Goal{}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &goals); err != nil {
		return returnError(err)
	}

	body, err := json.Marshal(goals)
	if err != nil {
		return returnError(err)
	}

	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(handleGoalGetAllEvent)
}
