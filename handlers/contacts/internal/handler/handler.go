// Package handler provides the Lambda function implementation.
package handler

import (
	books "bookoftheday/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SESv2ListContactsPaginatorAPI simplifies retrieval of pages of contacts.
type SESv2ListContactsPaginatorAPI interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*sesv2.Options)) (*sesv2.ListContactsOutput, error)
}

// SESv2NewListContactsPaginatorAPI allows creating new Paginators for ListContacts operations.
type SESv2NewListContactsPaginatorAPI func(
	client sesv2.ListContactsAPIClient, params *sesv2.ListContactsInput, optFns ...func(*sesv2.ListContactsPaginatorOptions),
) SESv2ListContactsPaginatorAPI

// SQSSendMessageAPI allows sending a message to an SQS queue.
type SQSSendMessageAPI interface {
	SendMessage(ctx context.Context,
		params *sqs.SendMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// Handler provides the Lambda implementation list contacts and send them to an SQS queue.
type Handler struct {
	lcAPI           sesv2.ListContactsAPIClient
	newLCPaginator  SESv2NewListContactsPaginatorAPI
	contactListName string
	smAPI           SQSSendMessageAPI
	queueURL        string
	rng             *rand.Rand
}

// New creates a new Handler instance.
func New(
	lcAPI sesv2.ListContactsAPIClient,
	newLCPaginator SESv2NewListContactsPaginatorAPI,
	contactListName string, smAPI SQSSendMessageAPI,
	queueURL string,
	rng *rand.Rand,
) *Handler {
	return &Handler{
		lcAPI:           lcAPI,
		newLCPaginator:  newLCPaginator,
		contactListName: contactListName,
		smAPI:           smAPI,
		queueURL:        queueURL,
		rng:             rng,
	}
}

type result struct {
	contactEmail string
	messageID    *string
	err          error
}

func sendContact(ctx context.Context, contact sestypes.Contact, book books.BestSellerBook, api SQSSendMessageAPI, queueURL string) (*sqs.SendMessageOutput, error) {
	mb := books.SQSBookMessageBody{ContactEmail: *contact.EmailAddress, Book: book}
	b, err := json.Marshal(mb)
	if err != nil {
		return nil, fmt.Errorf("could not marshal message body: %w", err)
	}

	return api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl: &queueURL,
		MessageAttributes: map[string]sqstypes.MessageAttributeValue{
			"LastUpdatedTimestamp": {
				DataType:    aws.String("String"),
				StringValue: aws.String(contact.LastUpdatedTimestamp.String()),
			},
			"ContactEmail": {
				DataType:    aws.String("String"),
				StringValue: contact.EmailAddress,
			},
		},
		MessageBody: aws.String(string(b)),
	})
}

// EnqueueContacts gets the list of contacts and sends them to an SQS queue.
func (h *Handler) EnqueueContacts(bookList []books.BestSellerBook) error {
	if len(bookList) == 0 {
		return errors.New("cannot process input - books list was empty")
	}

	p := h.newLCPaginator(h.lcAPI, &sesv2.ListContactsInput{
		ContactListName: &h.contactListName,
	})

	results := make(chan result)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for p.HasMorePages() {
		out, err := p.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("error getting page of contacts: %w", err)
		}

		wg.Add(len(out.Contacts))
		for _, c := range out.Contacts {
			n := h.rng.Intn(len(bookList))
			book := bookList[n]
			go func(contact sestypes.Contact) {
				defer wg.Done()
				out, err := sendContact(ctx, contact, book, h.smAPI, h.queueURL)
				result := result{err: err, contactEmail: *contact.EmailAddress}
				if out != nil {
					result.messageID = out.MessageId
				}
				results <- result
			}(c)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	errs := 0
	for r := range results {
		if r.err != nil {
			errs++
			log.Printf("error sending contact %s to SQS:\n %v", r.contactEmail, r)
		}
	}

	if errs != 0 {
		return errors.New("there were errors sending SQS messages, check log output")
	}

	return nil
}
