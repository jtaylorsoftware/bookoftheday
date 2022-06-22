package main

import (
	"context"
	"log"
	"os"
	"send-email/internal/handler"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln("configuration error: " + err.Error())
	}

	sesClient := sesv2.NewFromConfig(cfg)

	h := handler.New(handler.Config{
		SendEmailAPI:     sesClient,
		ContactListName:  os.Getenv("CONTACT_LIST_NAME"),
		ConfigurationSet: os.Getenv("CONFIGURATION_SET"),
		TopicName:        os.Getenv("TOPIC_NAME"),
		FromEmailAddress: os.Getenv("FROM_EMAIL_ADDR"),
	})
	lambda.Start(h.SendEmailWithBook)
}
