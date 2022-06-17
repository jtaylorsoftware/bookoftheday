package main

import (
	"context"
	"log"
	"os"

	"lists/internal/handler"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln("configuration error: " + err.Error())
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	// Must wrap library method to satisfy the handler.DynamoDBScanPaginatorAPI return type.
	p := func(client dynamodb.ScanAPIClient, params *dynamodb.ScanInput, optFns ...func(*dynamodb.ScanPaginatorOptions)) handler.DynamoDBScanPaginatorAPI {
		return dynamodb.NewScanPaginator(client, params, optFns...)
	}
	h := handler.New(ddbClient, p, os.Getenv("LISTS_TABLE_NAME"))

	lambda.Start(h.GetBestSellerLists)
}
