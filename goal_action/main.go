package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
)

type Request events.APIGatewayProxyRequest
type Response events.APIGatewayProxyResponse

var GOAL_TABLE = "xeffect_goals"

type Goal struct {
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

type GoalAction struct {
	Type string `json:"action" validate:"required"`
}

type GoalMarkCompleted struct {
	IsCompleted *bool  `json:"is_completed" validate:"required"`
	Date        string `json:"date" validate:"required,datetime=2006-01-02"`
	// StreakStartDate string `json:"streak_start_date"`
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

func handleGoalActionEvent(ctx context.Context, event Request) (Response, error) {
	goalId := event.PathParameters["goalId"]

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

	var action GoalAction
	if err := json.Unmarshal(body, &action); err != nil {
		return returnError(err)
	}

	validate := validator.New()
	if err := validate.Struct(action); err != nil {
		return returnError(err)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-2"))
	if err != nil {
		return returnError(err)
	}

	client := dynamodb.NewFromConfig(cfg)

	switch action.Type {
	case "mark_completed":
		err = goalMarkCompleted(ctx, client, goalId, body)
	}

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

func getGoalStreak(ctx context.Context, client *dynamodb.Client, id string, streakDate string) (Goal, error) {
	input := &dynamodb.GetItemInput{
		TableName: &GOAL_TABLE,
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ProjectionExpression: aws.String("#streaksMap.#streakDate"),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap": "Streaks",
			"#streakDate": streakDate,
		},
	}

	result, err := client.GetItem(ctx, input)
	if err != nil {
		return Goal{}, err
	}

	var goal Goal
	if err := attributevalue.UnmarshalMap(result.Item, &goal); err != nil {
		return Goal{}, err
	}

	return goal, nil
}

func getGoal(ctx context.Context, client *dynamodb.Client, id string) (Goal, error) {
	input := &dynamodb.GetItemInput{
		TableName: &GOAL_TABLE,
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
	}

	result, err := client.GetItem(ctx, input)
	if err != nil {
		return Goal{}, err
	}

	var goal Goal
	if err := attributevalue.UnmarshalMap(result.Item, &goal); err != nil {
		return Goal{}, err
	}

	return goal, nil
}

func parseDate(date string) (time.Time, error) {
	return time.Parse("2006-01-02", date)
}

func daysBetweenDates(earlierDate string, laterDate string) (int, error) {
	a, err := parseDate(earlierDate)
	if err != nil {
		return 0, err
	}

	b, err := parseDate(laterDate)
	if err != nil {
		return 0, err
	}

	return int(b.Sub(a).Hours() / 24), nil
}

func dateInStreak(streakDate string, streakLength int, date string) (bool, error) {
	days, err := daysBetweenDates(streakDate, date)
	if err != nil {
		return false, err
	}

	inStreak := false
	if days < streakLength {
		inStreak = true
	}

	return inStreak, nil
}

func mergeStreaks(ctx context.Context, client *dynamodb.Client, id string, streakDateA string, streakDateB string, streakDateIndexB int, streakLengthB int) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(GOAL_TABLE),
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ReturnValues:     types.ReturnValueNone,
		UpdateExpression: aws.String(fmt.Sprintf("ADD #streaksMap.#streakDate.#streakLength :inc REMOVE #streaksMap.#oldStreakDate, #streakDates[%d]", streakDateIndexB)),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap":    "Streaks",
			"#streakDate":    streakDateA,
			"#streakLength":  "Length",
			"#oldStreakDate": streakDateB,
			"#streakDates":   "StreakDates",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", streakLengthB),
			},
		},
	}
	_, err := client.UpdateItem(ctx, input)
	if err != nil {
		return err
	}

	return err
}

func updateStreak(ctx context.Context, client *dynamodb.Client, id string, streakDate string, positive bool) error {
	increment := "1"
	if !positive {
		increment = "-1"
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(GOAL_TABLE),
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ReturnValues:     types.ReturnValueNone,
		UpdateExpression: aws.String("ADD #streaksMap.#streakDate.#streakLength :inc"),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap":   "Streaks",
			"#streakDate":   streakDate,
			"#streakLength": "Length",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{
				Value: increment,
			},
		},
	}
	_, err := client.UpdateItem(ctx, input)

	return err
}

func removeFirstDayFromStreak(ctx context.Context, client *dynamodb.Client, id string, oldStreakDate string, oldStreak GoalStreak, oldStreakDateIndex int) error {
	date, err := parseDate(oldStreakDate)
	if err != nil {
		return err
	}

	date = date.AddDate(0, 0, 1)
	newStreakDate := date.Format("2006-01-02")

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(GOAL_TABLE),
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ReturnValues:     types.ReturnValueNone,
		UpdateExpression: aws.String(fmt.Sprintf("SET #streaksMap.#newStreakDate = :newStreak, #streakDates[%d] = :newStreakDate REMOVE #streaksMap.#oldStreakDate", oldStreakDateIndex)),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap":    "Streaks",
			"#oldStreakDate": oldStreakDate,
			"#newStreakDate": newStreakDate,
			"#streakDates":   "StreakDates",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":newStreak": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"Length": &types.AttributeValueMemberN{
						Value: fmt.Sprintf("%d", oldStreak.Length-1),
					},
				},
			},
			":newStreakDate": &types.AttributeValueMemberS{
				Value: newStreakDate,
			},
		},
	}
	_, err = client.UpdateItem(ctx, input)

	return err
}

func bringStreakForwardOneDay(ctx context.Context, client *dynamodb.Client, id string, oldStreakDate string, newStreakDate string, oldStreak GoalStreak, oldStreakDateIndex int) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(GOAL_TABLE),
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ReturnValues:     types.ReturnValueNone,
		UpdateExpression: aws.String(fmt.Sprintf("SET #streaksMap.#newStreakDate = :newStreak, #streakDates[%d] = :newStreakDate REMOVE #streaksMap.#oldStreakDate", oldStreakDateIndex)),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap":    "Streaks",
			"#newStreakDate": newStreakDate,
			"#oldStreakDate": oldStreakDate,
			"#streakDates":   "StreakDates",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":newStreak": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"Length": &types.AttributeValueMemberN{
						Value: fmt.Sprintf("%d", oldStreak.Length+1),
					},
				},
			},
			":newStreakDate": &types.AttributeValueMemberS{
				Value: newStreakDate,
			},
		},
	}
	_, err := client.UpdateItem(ctx, input)

	return err
}

func startNewStreak(ctx context.Context, client *dynamodb.Client, id string, streakDate string, streakDates []string, streakDateIndex int) error {
	streaks := append(streakDates[:streakDateIndex+1], streakDates[streakDateIndex:]...)
	streaks[streakDateIndex] = streakDate

	s, err := attributevalue.MarshalList(streaks)
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(GOAL_TABLE),
		Key: map[string]types.AttributeValue{
			"uuid": &types.AttributeValueMemberS{
				Value: id,
			},
		},
		ReturnValues:     types.ReturnValueNone,
		UpdateExpression: aws.String("SET #streaksMap.#streakDate = :newStreak, #streakDates = :streakDates"),
		ExpressionAttributeNames: map[string]string{
			"#streaksMap":  "Streaks",
			"#streakDate":  streakDate,
			"#streakDates": "StreakDates",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":newStreak": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"Length": &types.AttributeValueMemberN{
						Value: "1",
					},
				},
			},
			":streakDates": &types.AttributeValueMemberL{
				Value: s,
			},
		},
	}
	_, err = client.UpdateItem(ctx, input)

	return err
}

func goalMarkCompleted(ctx context.Context, client *dynamodb.Client, id string, body []byte) error {
	var action GoalMarkCompleted
	if err := json.Unmarshal(body, &action); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(action); err != nil {
		return err
	}

	goal, err := getGoal(ctx, client, id)
	if err != nil {
		return err
	}

	var i int
	for i = 0; i < len(goal.StreakDates); i++ {
		index := goal.StreakDates[i]
		indexDate, err := parseDate(index)
		if err != nil {
			return err
		}

		actionDate, err := parseDate(action.Date)
		if err != nil {
			return err
		}

		// If the action date is further in the past than the index,
		// the in cannot be in that streak.
		if indexDate.After(actionDate) {
			continue
		}

		streak := goal.Streaks[index]
		inStreak, err := dateInStreak(index, streak.Length, action.Date)
		if err != nil {
			return err
		}

		// Option Matrix:
		// 0. Not In Streak & Is Not Complete - No Action.
		// 1. In Streak & Is Complete - No Action.
		// 2. In Streak & Is Not Complete - Split the streak and remove the non-complete
		// date.
		// 3. 1 After Streak & Is Complete - Modify the neighbour streak to include the new
		// completion.
		// 4. 1 Before Previous Streak & Is Complete - Modify the previous streak to start at the new date.
		// 5. Not In Streak & Is Completed - Create new Streak.

		// 0. The date is not within the streak, and is not completed, so no changes need to be made.
		if !inStreak && !*action.IsCompleted {
			return nil
		}

		if inStreak {
			if *action.IsCompleted {
				// 1. The goal is already complete on this date.
				return nil
			}

			// 2. Split the streak into two, and remove the completion of the specified
			// day.
			daysBetween, err := daysBetweenDates(index, action.Date)
			if err != nil {
				return err
			}

			// If the date to be removed is the last date in a streak.
			if daysBetween == streak.Length-1 {
				// Decrement the streak.
				return updateStreak(ctx, client, id, index, false)
			}

			// If the date to be removed is the first date in a streak.
			if daysBetween == 0 {
				return removeFirstDayFromStreak(ctx, client, id, index, goal.Streaks[index], i)
			}

			return nil
		}

		// Check whether incrementing this streak now requires the streak to be
		// merged into the next.
		var (
			previousIndex     string
			previousIndexDate time.Time
		)
		if i > 0 {
			previousIndex = goal.StreakDates[i-1]
			previousIndexDate, err = parseDate(previousIndex)
			if err != nil {
				return err
			}
		}

		// 3. If the a goal has been completed on the next available streak day,
		// then increment the streak.
		// nextDayInStreak := indexDate.Add(time.Hour * time.Duration(streak.Length))
		nextDayInStreak := indexDate.AddDate(0, 0, streak.Length)
		if actionDate.Equal(nextDayInStreak) && *action.IsCompleted {
			err := updateStreak(ctx, client, id, index, true)
			if err != nil {
				return err
			}

			// If the previous streak (the one that is the next most in the future)
			// falls on the day after the action date, then merge the two streaks.
			if i > 0 {
				if previousIndexDate.Sub(actionDate).Hours() == 24 {
					return mergeStreaks(ctx, client, id, index, previousIndex, i-1, goal.Streaks[previousIndex].Length)
				}
			}

			return nil
		}

		// 4. Check if the completion can be added to the start of the previous streak.
		if i > 0 {
			previousDateInLastStreak := previousIndexDate.AddDate(0, 0, -1)
			if actionDate.Equal(previousDateInLastStreak) {
				return bringStreakForwardOneDay(ctx, client, id, previousIndex, action.Date, goal.Streaks[previousIndex], i-1)
			}
		}

		// If this statement is reached then the completion matches no current streaks.
		// So a new streak should be created.
		break
	}

	// 5. Create new streak.
	err = startNewStreak(ctx, client, id, action.Date, goal.StreakDates, i)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(handleGoalActionEvent)
}
