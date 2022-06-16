package main

import (
	"context"
	"log"
	"os"
	"refresh-lists/internal/api"
	"refresh-lists/internal/handler"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var h *handler.Handler

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln("configuration error: " + err.Error())
	}

	ssmClient := ssm.NewFromConfig(cfg)
	gpInput := &ssm.GetParameterInput{
		Name:           aws.String(os.Getenv("SSM_PARAM_NAME")),
		WithDecryption: true,
	}

	gpOutput, err := ssmClient.GetParameter(context.TODO(), gpInput)
	if err != nil {
		log.Fatalln("could not get SSM parameter: " + err.Error())
	}

	api := api.NewNYTBooksAPI(*gpOutput.Parameter.Value, "https://api.nytimes.com/svc/books/v3/lists/names.json")

	ddbClient := dynamodb.NewFromConfig(cfg)

	h = handler.New(api, os.Getenv("LISTS_TABLE_NAME"), ddbClient)

	lambda.Start(h.RefreshBestSellerLists)
}
