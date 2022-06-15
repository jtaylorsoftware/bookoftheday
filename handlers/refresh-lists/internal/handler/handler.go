// Package handler provides the Lambda function implementation.
package handler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"refresh-lists/internal/books"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// BooksAPI enables requests to the NYT Books API.
type BooksAPI interface {
	GetBestSellerListNames() ([]books.BestSellerList, error)
}

// DynamoDBBatchWriteItemAPI provides a testable interface for using the
// DynamoDB BatchWriteItem command.
type DynamoDBBatchWriteItemAPI interface {
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
}

// Handler encapsulates the necessary state for the refresh Lambda.
type Handler struct {
	api       BooksAPI
	tableName string
	ddb       DynamoDBBatchWriteItemAPI
}

// New creates a new Handler instance.
func New(api BooksAPI, tableName string, ddb DynamoDBBatchWriteItemAPI) *Handler {
	return &Handler{api, tableName, ddb}
}

func marshalListItems(list []books.BestSellerList) ([]types.WriteRequest, error) {
	var reqs []types.WriteRequest
	for _, item := range list {
		m, err := attributevalue.MarshalMap(item)
		if err != nil {
			return reqs, fmt.Errorf("error marshalling to AttributeValue: %w", err)
		}
		reqs = append(reqs, types.WriteRequest{
			PutRequest: &types.PutRequest{Item: m},
		})
	}
	return reqs, nil
}

func (h *Handler) batchWriteRequests(reqs []types.WriteRequest) ([]types.WriteRequest, error) {
	const ItemsPerBatch = 25
	nb := int(math.Ceil(float64(len(reqs)) / float64(ItemsPerBatch)))
	var unprocessed []types.WriteRequest
	for b := 0; b < nb; b++ {
		start := b * ItemsPerBatch
		stop := start + ItemsPerBatch
		if stop > len(reqs) {
			stop = len(reqs)
		}
		bwInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				h.tableName: reqs[start:stop],
			},
		}
		bwOutput, err := h.ddb.BatchWriteItem(context.TODO(), bwInput)
		if err != nil {
			return unprocessed, fmt.Errorf("error doing BatchWriteItem: %w", err)
		}

		if len(bwOutput.UnprocessedItems) != 0 {
			unprocessed = append(unprocessed, bwOutput.UnprocessedItems[h.tableName]...)
		}
	}

	return unprocessed, nil
}

func (h *Handler) batchWriteList(list []books.BestSellerList) ([]types.WriteRequest, error) {
	reqs, err := marshalListItems(list)
	if err != nil {
		return nil, err
	}

	return h.batchWriteRequests(reqs)
}

// RefreshBestSellerLists fetches the latest Best Seller list names
// from the Books API and stores them in a DynamoDB table.
func (h *Handler) RefreshBestSellerLists() error {
	list, err := h.api.GetBestSellerListNames()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return errors.New("books API returned empty list")
	}

	unprocessed, err := h.batchWriteList(list)
	if err != nil {
		return err
	}

	backoff := 1 * time.Second
	max := 16 * time.Second
	for len(unprocessed) != 0 && backoff < max {
		time.Sleep(backoff)
		backoff *= 2
		unprocessed, err = h.batchWriteRequests(unprocessed)
		var pte types.ProvisionedThroughputExceededException
		if !errors.Is(err, &pte) {
			return err
		}
	}

	return err
}
