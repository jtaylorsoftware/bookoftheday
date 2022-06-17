package handler

import (
	books "bookoftheday/types"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
)

type dummyScanAPIClient struct{}

func (d *dummyScanAPIClient) Scan(context.Context, *dynamodb.ScanInput, ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return nil, nil
}

type stubDynamoDBScanPaginatorAPI struct {
	np           int
	p            int
	pages        [][]books.BestSellerList
	err          error
	pageErrorNum int
}

func (f *stubDynamoDBScanPaginatorAPI) HasMorePages() bool {
	return f.p != f.np
}

func (f *stubDynamoDBScanPaginatorAPI) NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	if f.err != nil && f.p == f.pageErrorNum {
		return &dynamodb.ScanOutput{}, f.err
	}

	t := f.p
	f.p++

	p := f.pages[t]
	items := []map[string]types.AttributeValue{}
	for _, v := range p {
		item, _ := attributevalue.MarshalMap(v)
		items = append(items, item)
	}

	return &dynamodb.ScanOutput{Count: int32(len(p)), Items: items}, nil
}

type mockDynamoDBNewScanPaginatorAPIProvider struct {
	*testing.T
	f *stubDynamoDBScanPaginatorAPI
}

func (m *mockDynamoDBNewScanPaginatorAPIProvider) mockDynamoDBNewScanPaginatorAPI(
	client dynamodb.ScanAPIClient, params *dynamodb.ScanInput, optFns ...func(*dynamodb.ScanPaginatorOptions),
) DynamoDBScanPaginatorAPI {
	if params == nil {
		m.Fatalf("NewScanPaginator: got nil params")
	}
	if *params.TableName != TableName {
		m.Fatalf("NewScanPaginator: got params.TableName with value %s; expected %s", *params.TableName, TableName)
	}

	return m.f
}

const TableName = "TABLE"

func TestHandler(t *testing.T) {
	testCases := []struct {
		numPages     int
		countPerPage int
		err          error
		pageErrorNum int // number of page to cause an error (if error non-nil)
	}{
		{0, 0, nil, 0}, {0, 0, errors.New("error"), 0}, {1, 10, nil, 0}, {1, 50, nil, 0},
		{2, 10, errors.New("error"), 1}, {2, 50, nil, 0},
		{3, 50, nil, 0}, {3, 20, errors.New("error"), 1}, {3, 20, errors.New("error"), 3},
	}

	f := fuzz.New()

	for _, tc := range testCases {
		errDesc := ""
		if tc.err != nil {
			errDesc = fmt.Sprintf(" with error %v at page %d", tc.err, tc.pageErrorNum)
		}
		t.Run(fmt.Sprintf("%d pages %d per page%s", tc.numPages, tc.countPerPage, errDesc), func(t *testing.T) {
			pages := make([][]books.BestSellerList, tc.numPages)
			for i := range pages {
				for j := 0; j < tc.countPerPage; j++ {
					pages[i] = append(pages[i], books.BestSellerList{})
					f.Fuzz(&pages[i][j])
				}
			}

			f := &stubDynamoDBScanPaginatorAPI{
				np:           tc.numPages,
				p:            0,
				pages:        pages,
				err:          tc.err,
				pageErrorNum: tc.pageErrorNum,
			}
			m := &mockDynamoDBNewScanPaginatorAPIProvider{t, f}
			h := New(&dummyScanAPIClient{}, m.mockDynamoDBNewScanPaginatorAPI, TableName)

			out, err := h.GetBestSellerLists()

			if err != nil && !errors.Is(err, tc.err) {
				t.Errorf("got error %v; expected %v", err, tc.err)
			}

			if tc.err != nil && f.p != tc.pageErrorNum {
				t.Errorf("handler continued processing despite error at page %d", tc.pageErrorNum)
			}

			if tc.err == nil && f.p != tc.numPages {
				t.Errorf("handler did not process all pages: got %d; num pages: %d", f.p, tc.numPages)
			}

			if tc.err == nil {
				outLen := len(out.Lists)
				expLen := tc.numPages * tc.countPerPage
				if outLen != out.Count {
					t.Fatalf("output.Count mismatch len(output.List): got %d; expected %d", out.Count, outLen)
				}

				if outLen != expLen {
					t.Fatalf("output had wrong len: got %d; expected %d", outLen, expLen)
				}

				for i, p := range pages {
					s := i * tc.countPerPage
					e := s + tc.countPerPage
					if !cmp.Equal(out.Lists[s:e], p) {
						t.Errorf("segment [%d:%d] of out.Lists was not equal to input page %d", s, e, i)
					}
				}
			}
		})
	}
}
