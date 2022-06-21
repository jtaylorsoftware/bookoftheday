// Package handler provides the Lambda function implementation.
package handler

import (
	"bookoftheday/types"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoDBQueryPaginatorAPI is a convenience wrapper over DynamoDB query operations and is unit-testable.
type DynamoDBQueryPaginatorAPI interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// DynamoDBNewQueryPaginatorAPI is a type that allows creating instances of DynamoDBQueryPaginatorAPI.
type DynamoDBNewQueryPaginatorAPI func(
	client dynamodb.QueryAPIClient, params *dynamodb.QueryInput, optFns ...func(*dynamodb.QueryPaginatorOptions),
) DynamoDBQueryPaginatorAPI

// Handler provides the state and implementation of the main Lambda function.
type Handler struct {
	queryClient       dynamodb.QueryAPIClient
	newQueryPaginator DynamoDBNewQueryPaginatorAPI
	tableName         string
}

// New creates a new Handler instance.
func New(qc dynamodb.QueryAPIClient, nqp DynamoDBNewQueryPaginatorAPI, tableName string) *Handler {
	return &Handler{
		qc,
		nqp,
		tableName,
	}
}

// BestSellerBooksResponse contains the response data from calling GetBooksOnDateInList successfully.
type BestSellerBooksResponse struct {
	Count       *int                   `json:"count,omitempty"`
	Attribution string                 `json:"attribution,omitempty"`
	Books       []types.BestSellerBook `json:"books,omitempty"`
	Errors      []ErrorInfo            `json:"errors,omitempty"`
}

// ErrorInfo contains information about errors in a request that resulted in an invalid response.
type ErrorInfo struct {
	Field    string `json:"field,omitempty"`
	Message  string `json:"message,omitempty"`
	Location string `json:"location"`
}

var dateRegexp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var listRegexp = regexp.MustCompile(`^[a-zA-Z]+(-[a-zA-Z]+)*$`)

// defaultLimit specifies the hard limit when there is no user-specified limit, or
// limit is greater than this value.
const defaultLimit int32 = 31

type requestParams struct {
	limit      int32
	date       *string
	list       *string
	dateOffset *string
}

func (h *Handler) GetBooksOnDateInList(req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	input, errors := validateReq(req)
	if len(errors) != 0 {
		return response(400, BestSellerBooksResponse{Errors: errors})
	}

	qInput := &dynamodb.QueryInput{
		TableName: &h.tableName,
		Limit:     &input.limit,
	}

	err := buildQuery(input, qInput)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("error building query: %w", err)
	}

	p := h.newQueryPaginator(h.queryClient, qInput)

	books := []types.BestSellerBook{}
	for p.HasMorePages() {
		out, err := p.NextPage(context.TODO())
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("could not query books: %w", err)
		}

		if out.Count != 0 {
			var data []types.BestSellerBook
			err := attributevalue.UnmarshalListOfMaps(out.Items, &data)
			if err != nil {
				return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("could not unmarshal books: %w", err)
			}
			books = append(books, data...)
		}
	}

	count := clampMax(len(books), int(input.limit))
	books = books[:count]
	return response(200, BestSellerBooksResponse{
		Count:       &count,
		Attribution: Attribution,
		Books:       books,
		Errors:      nil,
	})
}

const Attribution = "Data provided by The New York Times: https://developer.nytimes.com"

func buildQuery(input requestParams, query *dynamodb.QueryInput) error {
	var kceList *expression.KeyConditionBuilder
	var kceDate *expression.KeyConditionBuilder
	if input.list != nil {
		kce := expression.Key("ListEncodedName").Equal(expression.Value(*input.list))
		kceList = &kce
	}

	// Assuming input has already been sanitized so only one of date or date-offset is set
	if input.dateOffset != nil {
		kce := expression.Key("DateSelected").GreaterThanEqual(expression.Value(*input.dateOffset))
		kceDate = &kce
	} else if input.date != nil {
		kce := expression.Key("DateSelected").Equal(expression.Value(*input.date))
		kceDate = &kce
	}

	var kce expression.KeyConditionBuilder
	if kceList != nil {
		kce = *kceList
		if kceDate != nil {
			kce = kce.And(*kceDate)
		}
	} else {
		// Assuming input has already been sanitized so one of list or date is set
		kce = *kceDate
		query.IndexName = aws.String("DateSelectedIndex")
	}

	e, err := expression.NewBuilder().WithKeyCondition(kce).Build()
	if err != nil {
		return err
	}

	query.KeyConditionExpression = e.KeyCondition()
	query.ExpressionAttributeNames = e.Names()
	query.ExpressionAttributeValues = e.Values()
	query.FilterExpression = e.Filter()
	return nil
}

func validateReq(req events.APIGatewayV2HTTPRequest) (requestParams, []ErrorInfo) {
	reqInput := requestParams{}
	var errors []ErrorInfo

	if qLimit, ok := req.QueryStringParameters["limit"]; ok {
		if iLimit, err := strconv.ParseInt(qLimit, 10, 32); err == nil && iLimit >= 1 {
			reqInput.limit = int32(clampMax(int(iLimit), int(defaultLimit)))
		} else {
			errors = append(errors, ErrorInfo{"limit", "limit must be an integer >= 1", "query"})
		}
	} else {
		reqInput.limit = defaultLimit
	}

	if qDate, ok := req.QueryStringParameters["date"]; ok {
		if dateRegexp.MatchString(qDate) {
			reqInput.date = &qDate
		} else {
			errors = append(errors, ErrorInfo{"date", "date must be in the format yyyy-MM-dd", "query"})
		}
	}

	if qList, ok := req.QueryStringParameters["list"]; ok {
		if listRegexp.MatchString(qList) {
			reqInput.list = &qList
		} else {
			errors = append(errors, ErrorInfo{"list", fmt.Sprintf("list must be in the format %s", listRegexp.String()), "query"})
		}
	}

	if qDateOffset, ok := req.QueryStringParameters["date-offset"]; ok {
		if dateRegexp.MatchString(qDateOffset) {
			reqInput.dateOffset = &qDateOffset
		} else {
			errors = append(errors, ErrorInfo{"date-offset", "date-offset must be in the format yyyy-MM-dd", "query"})
		}
	}

	if reqInput.date != nil && reqInput.dateOffset != nil {
		errors = append(errors, ErrorInfo{Message: "only one of date or date-offset can be specified", Location: "query"})
	}

	if reqInput.date == nil && reqInput.list == nil {
		errors = append(errors, ErrorInfo{Message: "either list or date must be specified", Location: "query"})
	}

	return reqInput, errors
}

func clampMax(v, max int) int {
	return int(math.Min(float64(v), float64(max)))
}

func response(status int, body BestSellerBooksResponse) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		err = fmt.Errorf("error marshalling response body: %w", err)
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: string(b),
	}, err
}
