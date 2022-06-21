package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"random-book/internal/api"
	"random-book/internal/handler"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

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

	api := api.NewNYTBooksAPI(*gpOutput.Parameter.Value, "https://api.nytimes.com/svc/books/v3/lists/%s/%s.json")

	ddbClient := dynamodb.NewFromConfig(cfg)

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	h := handler.New(api, ddbClient, os.Getenv("BOOKS_TABLE_NAME"), r)
	lambda.Start(h.GetRandomBestSellerBook)
}
