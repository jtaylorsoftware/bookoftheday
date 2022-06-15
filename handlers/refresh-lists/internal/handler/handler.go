// Package handler provides the Lambda function implementation.
package handler

import (
	"context"
	"refresh-lists/internal/books"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// BooksAPI enables requests to the NYT Books API.
type BooksAPI interface {
	GetBestSellerListNames() ([]books.BestSellerList, error)
}

// DynamoDBPutItemAPI provides a testable interface for using the DynamoDB PutItem
// command.
type DynamoDBPutItemAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// Handler encapsulates the necessary state for the refresh Lambda.
type Handler struct {
	api BooksAPI
	ddb DynamoDBPutItemAPI
}

// New creates a new Handler instance.
func New(api BooksAPI, ddb DynamoDBPutItemAPI) *Handler {
	return &Handler{api, ddb}
}

// RefreshBestSellerLists fetches the latest Best Seller list names
// from the Books API and stores them in a DynamoDB table.
func (h *Handler) RefreshBestSellerLists() error {
	// list, err := h.api.GetBestSellerListNames()
	// fmt.Println(list)
	// return err
	return nil
}
