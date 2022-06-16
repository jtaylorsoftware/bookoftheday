package main

import (
	"context"
	"log"
	"os"
	"subscribe/internal/handler"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln("configuration error: " + err.Error())
	}

	client := sesv2.NewFromConfig(cfg)

	clName := os.Getenv("CONTACT_LIST_NAME")

	h := handler.New(client, clName)

	lambda.Start(h.Subscribe)
}
