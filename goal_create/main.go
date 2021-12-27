package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Request events.APIGatewayProxyRequest
type Response events.APIGatewayProxyResponse

const GOAL_TABLE = "xeffect_goals"

type Goal struct {
	Title      string `json:"title" validate:"required"`
	Motivation string `json:"motivation" validate:"required"`
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

func handleGoalCreationEvent(ctx context.Context, event Request) (Response, error) {
	var body []byte
	if event.IsBase64Encoded {
		var err error
		body, err = base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return returnError(err)
		}
	} else {
		body = []byte(event.Body)
	}

	contentType := event.Headers["content-type"]
	if contentType != "application/json" {
		headers, _ := json.Marshal(event.Headers)
		return returnError(fmt.Errorf("'%s' is not a supported Content-Type.\n%s", contentType, string(headers)))
	}

	var goal Goal
	if err := json.Unmarshal(body, &goal); err != nil {
		return returnError(err)
	}

	validate := validator.New()
	if err := validate.Struct(goal); err != nil {
		return returnError(err)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-2"))
	if err != nil {
		return returnError(err)
	}

	client := dynamodb.NewFromConfig(cfg)

	input := &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: uuid.New().String(),
			},
			"Title": &types.AttributeValueMemberS{
				Value: goal.Title,
			},
			"Motivation": &types.AttributeValueMemberS{
				Value: goal.Motivation,
			},
		},
		TableName: aws.String(GOAL_TABLE),
	}

	_, err = client.PutItem(ctx, input)
	if err != nil {
		return returnError(err)
	}

	return Response{
		StatusCode: 201,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func main() {
	lambda.Start(handleGoalCreationEvent)
}
