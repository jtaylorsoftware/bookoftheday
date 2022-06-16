// Package handler provides the Lambda function implementation.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESv2CreateContactAPI allows creating a new SES contact.
type SESv2CreateContactAPI interface {
	CreateContact(ctx context.Context, params *sesv2.CreateContactInput, optFns ...func(*sesv2.Options)) (*sesv2.CreateContactOutput, error)
}

type Handler struct {
	ses             SESv2CreateContactAPI
	contactListName string
}

// New creates an instance of Handler that will subscribe clients to `contactListName` by creating a contact.
func New(ses SESv2CreateContactAPI, contactListName string) *Handler {
	return &Handler{ses, contactListName}
}

// Subscribe creates a contact for the email associated with the request.
func (h *Handler) Subscribe(req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	email, ok := req.QueryStringParameters["email"]
	if !ok || len(email) == 0 {
		return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusBadRequest}, nil
	}

	// TODO
	// - Keep a table of subscribers separate from SES and generate verification link?
	// - Allow subscribing based on list names (use AttributesData or DDB)
	_, err := h.ses.CreateContact(context.TODO(), &sesv2.CreateContactInput{
		ContactListName: aws.String(h.contactListName),
		EmailAddress:    aws.String(email),
		TopicPreferences: []types.TopicPreference{
			{
				TopicName:          aws.String("Books"),
				SubscriptionStatus: types.SubscriptionStatusOptOut,
			},
		},
	})
	if err != nil {
		log.Printf("error in CreateContact: %v", err)
		var alreadyExists *types.AlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusConflict}, nil
		} else {
			return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusInternalServerError}, fmt.Errorf("error creating contact: %w", err)
		}
	}

	return events.APIGatewayV2HTTPResponse{StatusCode: 200}, nil
}
