package main

import (
	"context"
	"fmt"
	"time"

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
	Title      string                `json:"title" validate:"required"`
	Motivation string                `json:"motivation" validate:"required"`
	Streaks    map[string]GoalStreak `json:"streaks" validate:"required"`
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

func handleGoalGetCompletedEvent(ctx context.Context, event Request) (Response, error) {
	goalId := event.PathParameters["goalId"]
	date := event.PathParameters["date"]

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

	completed := false
	for streakDate, streak := range goal.Streaks {
		streakDate, err := time.Parse("2006-01-02", streakDate)
		if err != nil {
			return returnError(err)
		}

		date, err := time.Parse("2006-01-02", date)
		if err != nil {
			return returnError(err)
		}

		daysBetween := int(date.Sub(streakDate).Hours() / 24)

		// If the days between the date and the streak date is less than or equal to
		// the stream length, then the date was completed.
		if daysBetween >= 0 && daysBetween < streak.Length {
			completed = true
			break
		}
	}

	return Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: fmt.Sprintf("{\"%s\": %t}", date, completed),
	}, nil
}

func main() {
	lambda.Start(handleGoalGetCompletedEvent)
}
