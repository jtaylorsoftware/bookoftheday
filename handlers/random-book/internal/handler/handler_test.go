package handler

import (
	books "bookoftheday/types"
	"context"
	"random-book/internal/api"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	fuzz "github.com/google/gofuzz"
)

type mockGetBooksInBestSellerListAPI struct {
	*testing.T
	books      api.BestSellerBookList
	list       string
	oldestDate string
	newestDate string
	err        error
}

func (m *mockGetBooksInBestSellerListAPI) GetBooksInListOnDate(list string, date string) (api.BestSellerBookList, error) {
	if list != m.list {
		m.Errorf("incorrect list name: got %s; expected %s", list, m.list)
	}

	o, _ := time.Parse(ymdLayout, m.oldestDate)
	n, _ := time.Parse(ymdLayout, m.newestDate)
	d, _ := time.Parse(ymdLayout, date)
	if !o.Before(d) || !d.Before(n) {
		m.Errorf("date argument was not in range: got %s; expected value between %s and %s", date, m.oldestDate, m.newestDate)
	}

	return m.books, m.err
}

type mockDynamoDBPutItemAPI struct {
	*testing.T
	input *dynamodb.PutItemInput
	err   error
}

func (m *mockDynamoDBPutItemAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if params == nil {
		m.Fatal("PutItem: got nil params; expected non-nil")
	}
	if diff := cmp.Diff(
		m.input,
		params,
		cmpopts.IgnoreUnexported(dynamodb.PutItemInput{}, types.AttributeValueMemberS{}, types.AttributeValueMemberN{}),
		cmpopts.IgnoreMapEntries(func(k string, v types.AttributeValue) bool {
			return k == "Expiration"
		}),
	); diff != "" {
		m.Errorf("fields mismatch in PutItemInput (-want +got):\n%s", diff)
	}
	return nil, m.err
}

const TableName = "Table"

func TestHandler(t *testing.T) {
	t.Run("creates correct API query and saves response in DynamoDB", func(t *testing.T) {
		oldest := "2010-01-01"
		newest := "2020-12-31"
		list := "hardcover-nonfiction"
		f := fuzz.New()
		var bl api.BestSellerBookList
		f.SkipFieldsWithPattern(regexp.MustCompile("Books")).Fuzz(&bl)
		bl.Books = make([]api.BestSellerBook, 1)
		f.Fuzz(&bl.Books[0])

		mAPI := &mockGetBooksInBestSellerListAPI{t, bl, list, oldest, newest, nil}

		b := bl.Books[0]
		book := books.BestSellerBook{
			ListEncodedName:   list,
			DateSelected:      time.Now().Format(ymdLayout),
			ListPublishedDate: bl.PublishedDate,
			ListDisplayName:   bl.DisplayName,
			ListUpdatePeriod:  bl.Updated,
			PrimaryISBN10:     b.PrimaryISBN10,
			PrimaryISBN13:     b.PrimaryISBN13,
			Title:             b.Title,
			Author:            b.Author,
			Publisher:         b.Publisher,
			Description:       b.Description,
			Rank:              b.Rank,
			AmazonProductURL:  b.AmazonProductURL,
			ImageURL:          b.ImageURL,
			ImageWidth:        b.ImageWidth,
			ImageHeight:       b.ImageHeight,
		}
		item, _ := attributevalue.MarshalMap(book)
		input := &dynamodb.PutItemInput{
			TableName: aws.String(TableName),
			Item:      item,
		}
		mDDB := &mockDynamoDBPutItemAPI{t, input, nil}

		h := New(mAPI, mDDB, TableName)
		now := time.Now().Unix()
		got, err := h.GetRandomBestSellerBook(books.BestSellerList{
			EncodedName:         list,
			OldestPublishedDate: oldest,
			NewestPublishedDate: newest,
		})

		if err != nil {
			t.Errorf("handler returned unexpected error: got %v; expected %v", err, nil)
		}

		if diff := cmp.Diff(book, got, cmpopts.IgnoreFields(books.BestSellerBook{}, "Expiration")); diff != "" {
			t.Errorf("fields mismatch in returned book (-want +got):\n%s", diff)
		}

		if got.Expiration == 0 || got.Expiration < now {
			t.Errorf("invalid expiration: got %d; expected value greater than %d", got.Expiration, now)
		}
	})
}
