package handler

import (
	"context"
	"fmt"
	"math"
	"refresh-lists/internal/books"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
)

type fakeBooksAPI struct {
	data []books.BestSellerList
}

func (f fakeBooksAPI) GetBestSellerListNames() ([]books.BestSellerList, error) {
	return f.data, nil
}

type mockDynamoDBBatchWriteItemAPI struct {
	t           *testing.T
	expected    []books.BestSellerList
	processed   int
	unprocessed map[string][]types.WriteRequest
	calls       int
}

const TableName = "TABLE"

func (md *mockDynamoDBBatchWriteItemAPI) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	md.calls++
	if params == nil {
		md.t.Fatal("got nil params; expected non-nil")
	}
	if len(params.RequestItems) == 0 {
		md.t.Fatal("got empty params.RequestItems")
	}
	for table, wrs := range params.RequestItems {
		if table != TableName {
			md.t.Fatalf("got table name %s; expected %s", table, TableName)
		}

		if len(wrs) == 0 {
			md.t.Fatal("got empty []WriteRequest")
		}

		for i, wr := range wrs {
			if wr.PutRequest == nil {
				md.t.Fatalf("got nil PutRequest for element %d", i)
			}
			var item books.BestSellerList
			err := attributevalue.UnmarshalMap(wr.PutRequest.Item, &item)
			if err != nil {
				md.t.Fatalf("got error unmarshalling item: %v", err)
			}

			// Use batch number to figure out which portion of the expected values we're in
			batch := int(math.Floor(float64(md.processed) / float64(25)))
			j := batch*25 + i
			if !cmp.Equal(item, md.expected[j]) {
				md.t.Fatalf("PutRequest.Item and actual were not equal at index %d: got %v; expected %v", j, item, md.expected[j])
			}
			// Keep track of how many items seen
			md.processed++
		}
	}

	out := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: md.unprocessed,
	}
	md.unprocessed = nil
	return out, nil
}

func TestHandler(t *testing.T) {
	// Simple cases for testing number of PutItem per BatchWrite
	testCases := []int{1, 24, 25, 26, 49, 52, 53}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("writes %d items to table", tc), func(t *testing.T) {
			items := make([]books.BestSellerList, tc)
			f := fuzz.New()
			for i := range items {
				f.Fuzz(&items[i])
			}

			md := &mockDynamoDBBatchWriteItemAPI{t: t, expected: items}
			h := New(fakeBooksAPI{items}, TableName, md)

			err := h.RefreshBestSellerLists()
			if err != nil {
				t.Fatalf("got non-nil error %v; expected nil", err)
			}

			// should be called once per batch (max 25 per batch)
			calls := int(math.Ceil(float64(tc) / float64(25)))
			if md.calls != calls {
				t.Fatalf("BatchWrite called %d times; expected %d times", md.calls, calls)
			}

			// ensure checked every item
			processed := len(items)
			if md.processed != processed {
				t.Fatalf("processed %d items; expected %d items", md.processed, processed)
			}
		})
	}

	t.Run("handles unprocessed items", func(t *testing.T) {
		var item books.BestSellerList
		f := fuzz.New()
		f.Fuzz(&item)

		items := []books.BestSellerList{item}
		unprocessed, _ := attributevalue.MarshalMap(item)

		md := &mockDynamoDBBatchWriteItemAPI{
			t:        t,
			expected: items,
			unprocessed: map[string][]types.WriteRequest{
				TableName: {{PutRequest: &types.PutRequest{Item: unprocessed}}},
			},
		}
		h := New(fakeBooksAPI{items}, TableName, md)

		err := h.RefreshBestSellerLists()
		if err != nil {
			t.Fatalf("got non-nil error %v; expected nil", err)
		}
		calls := 2
		if md.calls != calls {
			t.Fatalf("BatchWrite called %d times; expected %d times", md.calls, calls)
		}
		processed := 2 // items and unprocessed
		if md.processed != processed {
			t.Fatalf("processed %d items; expected %d items", md.processed, processed)
		}
	})
}
