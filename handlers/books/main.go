package main

import (
	"books/internal/handler"
	"context"
	"log"
	"os"

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
	p := func(client dynamodb.QueryAPIClient, params *dynamodb.QueryInput, optFns ...func(*dynamodb.QueryPaginatorOptions)) handler.DynamoDBQueryPaginatorAPI {
		return dynamodb.NewQueryPaginator(client, params, optFns...)
	}
	h := handler.New(ddbClient, p, os.Getenv("BOOKS_TABLE_NAME"))

	lambda.Start(h.GetBooksOnDateInList)
}
