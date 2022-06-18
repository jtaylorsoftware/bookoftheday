// Package handler provides the Lambda function implementation.
package handler

import (
	books "bookoftheday/types"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"random-book/internal/api"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// GetBooksInBestSellerListAPI allows querying the NYT API to get a list of best-selling books from
// a Best-Seller List published on a certain date.
type GetBooksInBestSellerListAPI interface {
	GetBooksInListOnDate(list string, date string) (api.BestSellerBookList, error)
}

// DynamoDBPutItemAPI provides a unit-testable interface to access the DynamoDB PutItem API.
type DynamoDBPutItemAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// Handler provides the Lambda implementation to get a random book given a Best-Seller list.
type Handler struct {
	api       GetBooksInBestSellerListAPI
	ddb       DynamoDBPutItemAPI
	tableName string
}

// New creates an instance of Handler.
func New(api GetBooksInBestSellerListAPI, ddb DynamoDBPutItemAPI, tableName string) *Handler {
	return &Handler{
		api,
		ddb,
		tableName,
	}
}

const ymdLayout = "2006-01-02"

// getRandomDateBetween calculates a random yyyy-MM-dd date string between two dates with the same format.
func getRandomDateBetween(oldest, newest string) (string, error) {
	o, err := time.Parse(ymdLayout, oldest)
	if err != nil {
		return "", fmt.Errorf("error parsing date to yyyy-MM-dd: %s", oldest)
	}
	n, err := time.Parse(ymdLayout, newest)
	if err != nil {
		return "", fmt.Errorf("error parsing date to yyyy-MM-dd: %s", newest)
	}

	days := int(math.Floor(n.Sub(o).Hours() / 24))
	rd := rand.Intn(days)
	date := o.Add(time.Duration(rd) * time.Hour * 24).Format(ymdLayout)
	return date, nil
}

// GetRandomBestSellerBook finds and persists a random book from a given Best-Seller List.
func (h *Handler) GetRandomBestSellerBook(list books.BestSellerList) (books.BestSellerBook, error) {
	date, err := getRandomDateBetween(list.OldestPublishedDate, list.NewestPublishedDate)
	if err != nil {
		return books.BestSellerBook{}, err
	}

	bl, err := h.api.GetBooksInListOnDate(list.EncodedName, date)
	if err != nil {
		return books.BestSellerBook{}, err
	}
	if len(bl.Books) == 0 {
		return books.BestSellerBook{}, errors.New("books API returned empty list")
	}

	b := bl.Books[rand.Intn(len(bl.Books))]
	bsb := books.BestSellerBook{
		ListEncodedName:  list.EncodedName,
		Date:             time.Now().Format(ymdLayout),
		ListDate:         bl.PublishedDate,
		ListDisplayName:  bl.DisplayName,
		ListUpdatePeriod: bl.Updated,
		PrimaryISBN10:    b.PrimaryISBN10,
		PrimaryISBN13:    b.PrimaryISBN13,
		Title:            b.Title,
		Author:           b.Author,
		Publisher:        b.Publisher,
		Rank:             b.Rank,
		AmazonProductURL: b.AmazonProductURL,
		ImageURL:         b.ImageURL,
		ImageWidth:       b.ImageWidth,
		ImageHeight:      b.ImageHeight,
		Expiration:       time.Now().AddDate(0, 1, 0).Unix(),
	}

	item, err := attributevalue.MarshalMap(bsb)
	if err != nil {
		return bsb, fmt.Errorf("could not marshal book value: %w", err)
	}

	_, err = h.ddb.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &h.tableName,
		Item:      item,
	}, func(o *dynamodb.Options) {
		o.Retryer = retry.AddWithMaxBackoffDelay(retry.NewStandard(), time.Second*8)
	})

	return bsb, err
}
