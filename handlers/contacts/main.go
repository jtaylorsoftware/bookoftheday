package main

import (
	"contacts/internal/handler"
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln("configuration error: " + err.Error())
	}

	sesClient := sesv2.NewFromConfig(cfg)
	newListContactsPaginator := func(
		client sesv2.ListContactsAPIClient, params *sesv2.ListContactsInput, optFns ...func(*sesv2.ListContactsPaginatorOptions),
	) handler.SESv2ListContactsPaginatorAPI {
		return sesv2.NewListContactsPaginator(client, params, optFns...)
	}

	sqsClient := sqs.NewFromConfig(cfg)

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	h := handler.New(sesClient, newListContactsPaginator, os.Getenv("CONTACT_LIST_NAME"), sqsClient, os.Getenv("EMAIL_QUEUE_URL"), r)
	lambda.Start(h.EnqueueContacts)
}
