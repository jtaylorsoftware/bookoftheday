// Package handler provides the Lambda function implementation.
package handler

import (
	books "bookoftheday/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESv2SendEmailAPI allows sending emails to contacts.
type SESv2SendEmailAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Handler provides the Lambda implementation list contacts and send them to an SQS queue.
type Handler struct {
	seAPI            SESv2SendEmailAPI
	contactListName  string
	configurationSet string
	topicName        string
	fromEmailAddr    string
}

// Config provides configuration options for a Handler.
type Config struct {
	SendEmailAPI     SESv2SendEmailAPI
	ContactListName  string
	ConfigurationSet string
	TopicName        string
	FromEmailAddress string
}

// New creates a new Handler instance.
func New(cfg Config) *Handler {
	return &Handler{
		seAPI:            cfg.SendEmailAPI,
		contactListName:  cfg.ContactListName,
		configurationSet: cfg.ConfigurationSet,
		topicName:        cfg.TopicName,
		fromEmailAddr:    cfg.FromEmailAddress,
	}
}

type failure struct {
	messageID *string
	err       error
}

func (h *Handler) sendEmail(ctx context.Context, msg events.SQSMessage) error {
	var body books.SQSBookMessageBody
	err := json.Unmarshal([]byte(msg.Body), &body)
	if err != nil {
		return fmt.Errorf("could not unmarshal body: %w", err)
	}

	txt, html := formatBodyContent(body.Book)
	_, err = h.seAPI.SendEmail(ctx, &sesv2.SendEmailInput{
		Destination: &sestypes.Destination{
			ToAddresses: []string{body.ContactEmail},
		},
		FromEmailAddress:     &h.fromEmailAddr,
		ConfigurationSetName: &h.configurationSet,
		ListManagementOptions: &sestypes.ListManagementOptions{
			ContactListName: &h.contactListName,
			TopicName:       &h.topicName,
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{Charset: aws.String(charset), Data: aws.String(subject)},
				Body: &sestypes.Body{
					Text: &sestypes.Content{Charset: aws.String(charset), Data: aws.String(txt)},
					Html: &sestypes.Content{Charset: aws.String(charset), Data: aws.String(html)},
				},
			},
		},
	})
	return err
}

// SendEmailWithBook gets a random book and emails it to the contact.
func (h *Handler) SendEmailWithBook(event events.SQSEvent) (events.SQSEventResponse, error) {
	failures := make(chan failure)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(event.Records))
	for _, msg := range event.Records {
		msg := msg
		go func() {
			defer wg.Done()

			err := h.sendEmail(ctx, msg)
			if err != nil {
				failures <- failure{err: err, messageID: &msg.MessageId}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(failures)
	}()

	batchItemFailures := []events.SQSBatchItemFailure{}
	for f := range failures {
		if f.err != nil {
			log.Printf("error sending email for MessageId %s: %v", *f.messageID, f.err)
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: *f.messageID})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}

const subject = "Book of the Day"
const charset = "UTF-8"

//	Parameters in order:
//		- Title
// 		- Author
// 		- Rank
//		- ListDisplayName
// 		- ListPublishedDate
//		- Description
//		- Publisher
// 		- PrimaryISBN10
// 		- PrimaryISBN13
//		- AmazonProductURL
const bodyText = `Book of the Day
Your Book of the Day is "%s" by "%s". It was rank %d for the list "%s" published %s.
Description: %s
Publisher: %s
ISBN10: %s
ISBN13: %s
Amazon: %s
Unsubscribe: {{amazonSESUnsubscribeUrl}}
`

//	TODO: Use template
//	Parameters in order:
//		- ImageURL
//		- Title
//		- ImageWidth
//		- ImageHeight
//		- Title
// 		- Author
// 		- Rank
//		- ListDisplayName
// 		- ListPublishedDate
//		- Description
//		- Publisher
// 		- PrimaryISBN10
// 		- PrimaryISBN13
//		- AmazonProductURL
const bodyHTML = `<html>
<head>
<style>
	img {
		border: 1px solid black;
		max-height: 350px;
		width: auto;
	}
</style>
</head>
<body>
	<h1>Book of the Day</h1>
	<img src="%s" alt="%s cover image" width="%d" height="%d">
	<p>Your Book of the Day is "%s" by "%s". It was rank %d for the list "%s" published %s.</p>
	<p>Description: %s</p>
	<p>Publisher: %s</p>
	<p><span>ISBN10: %s</span><br><span>ISBN13: %s</span></p>
	<p>Get it on Amazon: <a href="%s" target="_blank">here</a></p>
	<a href="{{amazonSESUnsubscribeUrl}}" target="_blank">Unsubscribe</a>
</body>
</html>
`

// formatBodyContent formats and returns the email body text and HTML for a given book.
func formatBodyContent(book books.BestSellerBook) (string, string) {
	txt := fmt.Sprintf(bodyText,
		book.Title,
		book.Author,
		book.Rank,
		book.ListDisplayName,
		book.ListPublishedDate,
		book.Description,
		book.Publisher,
		book.PrimaryISBN10,
		book.PrimaryISBN13,
		book.AmazonProductURL,
	)
	html := fmt.Sprintf(bodyHTML,
		book.ImageURL,
		book.Title,
		book.ImageWidth,
		book.ImageHeight,
		book.Title,
		book.Author,
		book.Rank,
		book.ListDisplayName,
		book.ListPublishedDate,
		book.Description,
		book.Publisher,
		book.PrimaryISBN10,
		book.PrimaryISBN13,
		book.AmazonProductURL,
	)
	return txt, html
}
