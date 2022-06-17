// Package handler provides the Lambda function implementation.
package handler

import (
	"context"
	"fmt"

	"bookoftheday/types"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoDBScanPaginatorAPI is a convenience wrapper over DynamoDB scan operations and is unit-testable.
type DynamoDBScanPaginatorAPI interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// DynamoDBNewScanPaginatorAPI is a type that allows creating instances of DynamoDBScanPaginatorAPI.
type DynamoDBNewScanPaginatorAPI func(
	client dynamodb.ScanAPIClient, params *dynamodb.ScanInput, optFns ...func(*dynamodb.ScanPaginatorOptions),
) DynamoDBScanPaginatorAPI

// Handler provides the state and implementation of the main Lambda function.
type Handler struct {
	scanClient       dynamodb.ScanAPIClient
	newScanPaginator DynamoDBNewScanPaginatorAPI
	tableName        string
}

// New creates a new Handler instance.
func New(sc dynamodb.ScanAPIClient, nsp DynamoDBNewScanPaginatorAPI, tableName string) *Handler {
	return &Handler{
		sc,
		nsp,
		tableName,
	}
}

// BestSellerListsResponse contains the response data from calling GetBestSellerLists successfully.
type BestSellerListsResponse struct {
	Count       int                    `json:"count"`
	Attribution string                 `json:"attribution"`
	Lists       []types.BestSellerList `json:"lists"`
}

const Attribution = "Data provided by The New York Times: https://developer.nytimes.com"

// GetBestSellerLists returns the complete collection of Best-Seller lists currently stored
// in the associated table.
func (h *Handler) GetBestSellerLists() (BestSellerListsResponse, error) {
	// A Paginator has to be made per-ScanInput so it's not a reusable resource and
	// instead per-request (although currently every request is identical).
	p := h.newScanPaginator(h.scanClient, &dynamodb.ScanInput{
		TableName: &h.tableName,
	})

	lists := []types.BestSellerList{}
	for p.HasMorePages() {
		out, err := p.NextPage(context.TODO())
		if err != nil {
			return BestSellerListsResponse{}, fmt.Errorf("could not get lists: %w", err)
		}

		if out.Count != 0 {
			var data []types.BestSellerList
			err := attributevalue.UnmarshalListOfMaps(out.Items, &data)
			if err != nil {
				return BestSellerListsResponse{}, fmt.Errorf("could not unmarshal lists: %w", err)
			}
			lists = append(lists, data...)
		}
	}

	return BestSellerListsResponse{
		Count:       len(lists),
		Attribution: Attribution,
		Lists:       lists,
	}, nil
}
