package handler

import (
	books "bookoftheday/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	fuzz "github.com/google/gofuzz"
)

func TestQueryParams(t *testing.T) {
	testCases := []struct {
		list       *string
		date       *string
		limit      *string
		dateOffset *string
	}{
		// combinations of optional params (where they're valid if present)
		{nil, nil, nil, nil},
		{nil, nil, str("21"), nil},
		{nil, str("2012-12-12"), nil, nil},
		{nil, str("2012-12-12"), str("12"), nil},
		{str("hardcover-nonfiction"), nil, nil, nil},
		{str("hardcover-nonfiction"), nil, str("101"), nil},
		{str("hardcover-nonfiction"), str("2012-12-12"), nil, nil},
		{str("hardcover-nonfiction"), nil, str("1"), str("2012-12-12")},

		// invalid params
		{str("hardcover-nonfiction"), str("2012-12-12"), str("1"), str("2012-12-12")}, // setting both date and date-offset
		{str("hardco213er&@1no55fiction"), str("2012-12-12"), str("1"), nil},
		{str("hardcover-nonfiction"), str("AAAA-BB-CC"), str("1"), nil},
		{str("hardcover-nonfiction"), str("2012-12-12"), str("A"), nil},
		{str("hardco213er&@1no55fiction"), str("AAAA-BB-CC"), str("A"), nil},
		{str("hardco213er&@1no55fiction"), nil, str("A"), str("AAAA$%adfBB#!%20CC")},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("query_list_%v_date_%v_limit_%v_offset_%v",
			describePStr(tc.list), describePStr(tc.date), describePStr(tc.limit), describePStr(tc.dateOffset)), func(t *testing.T) {
			query := map[string]string{}
			if tc.list != nil {
				query["list"] = *tc.list
			}
			if tc.date != nil {
				query["date"] = *tc.date
			}
			if tc.dateOffset != nil {
				query["date-offset"] = *tc.dateOffset
			}
			if tc.limit != nil {
				query["limit"] = *tc.limit
			}

			req := events.APIGatewayV2HTTPRequest{
				QueryStringParameters: query,
			}

			s := &stubDynamoDBQueryPaginatorAPI{
				np:           1,
				p:            0,
				pages:        [][]books.BestSellerBook{{books.BestSellerBook{}}},
				err:          nil,
				pageErrorNum: 0,
			}
			qp := &mockDynamoDBNewQueryPaginatorAPIProvider{t, s, query}

			h := New(&dummyQueryAPIClient{}, qp.mockDynamoDBNewScanPaginatorAPI, TableName)

			res, err := h.GetBooksOnDateInList(req)
			body := BestSellerBooksResponse{}
			_ = json.Unmarshal([]byte(res.Body), &body)

			if err != nil {
				t.Errorf("unexpected error: got %v; expected nil", err)
			}

			if tc.list != nil {
				errInd := containsError(body.Errors, "list")
				matches := listRegexp.MatchString(*tc.list)
				if matches && errInd != -1 {
					t.Errorf("unexpected error for field list: got %s; expected none", body.Errors[errInd].Message)
				} else if !matches && errInd == -1 {
					t.Errorf("missing error for field list")
				}
			}

			if tc.date != nil {
				errInd := containsError(body.Errors, "date")
				matches := dateRegexp.MatchString(*tc.date)
				if matches && errInd != -1 {
					t.Errorf("unexpected error for field date: got %s; expected none", body.Errors[errInd].Message)
				} else if !matches && errInd == -1 {
					t.Errorf("missing error for field date")
				}
			}

			if tc.dateOffset != nil {
				errInd := containsError(body.Errors, "date-offset")
				matches := dateRegexp.MatchString(*tc.dateOffset)
				if matches && errInd != -1 {
					t.Errorf("unexpected error for field dateOffset: got %s; expected none", body.Errors[errInd].Message)
				} else if !matches && errInd == -1 {
					t.Errorf("missing error for field dateOffset")
				}
			}

			if tc.list == nil && tc.date == nil && tc.dateOffset == nil {
				if len(body.Errors) == 0 {
					t.Errorf("missing error for nil list and date and date-offset")
				}
			}

			if tc.limit != nil {
				errInd := containsError(body.Errors, "limit")
				if _, err := strconv.ParseInt(*tc.limit, 10, 32); err == nil && errInd != -1 {
					t.Errorf("unexpected error for field limit: got %s; expected none", body.Errors[errInd].Message)
				} else if err != nil && errInd == -1 {
					t.Errorf("missing error for field limit")
				}
			}
		})
	}
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		numPages     int
		countPerPage int
		err          error
		pageErrorNum int // number of page to cause an error (if error non-nil)
		limit        int
	}{
		{0, 0, nil, 0, 1}, {0, 0, errors.New("error"), 0, 1}, {1, 10, nil, 0, 10}, {1, 50, nil, 0, 25},
		{2, 10, errors.New("error"), 1, 10}, {2, 50, nil, 0, 30},
		{3, 50, nil, 0, 40}, {3, 20, errors.New("error"), 1, 10}, {3, 20, errors.New("error"), 3, 10},
	}

	f := fuzz.New()
	for _, tc := range testCases {
		errDesc := ""
		if tc.err != nil {
			errDesc = fmt.Sprintf(" with error %v at page %d", tc.err, tc.pageErrorNum)
		}
		t.Run(fmt.Sprintf("%d pages %d per page limit %d%s", tc.numPages, tc.countPerPage, tc.limit, errDesc), func(t *testing.T) {
			pages := make([][]books.BestSellerBook, tc.numPages)
			for i := range pages {
				for j := 0; j < tc.countPerPage; j++ {
					pages[i] = append(pages[i], books.BestSellerBook{})
					f.Fuzz(&pages[i][j])
				}
			}
			query := map[string]string{"date": "2012-12-12"}
			query["limit"] = strconv.Itoa(tc.limit)
			qs := &stubDynamoDBQueryPaginatorAPI{
				np:           tc.numPages,
				p:            0,
				limit:        tc.limit,
				pages:        pages,
				err:          tc.err,
				pageErrorNum: tc.pageErrorNum,
			}
			m := &mockDynamoDBNewQueryPaginatorAPIProvider{t, qs, query}
			h := New(&dummyQueryAPIClient{}, m.mockDynamoDBNewScanPaginatorAPI, TableName)

			req := events.APIGatewayV2HTTPRequest{
				QueryStringParameters: query,
			}

			res, err := h.GetBooksOnDateInList(req)
			resBody := BestSellerBooksResponse{}
			_ = json.Unmarshal([]byte(res.Body), &resBody)

			if err != nil && !errors.Is(err, tc.err) {
				t.Errorf("got error %v; expected %v", err, tc.err)
			}

			if tc.err != nil && qs.p != tc.pageErrorNum {
				t.Errorf("handler continued processing despite error at page %d", tc.pageErrorNum)
			}

			if tc.err == nil && qs.p != tc.numPages {
				t.Errorf("handler did not process all pages: got %d; num pages: %d", qs.p, tc.numPages)
			}

			if tc.err == nil {
				outLen := len(resBody.Books)
				expLen := int(math.Min(float64(tc.numPages*tc.countPerPage), math.Min(float64(tc.limit), float64(defaultLimit))))
				if outLen != *resBody.Count {
					t.Fatalf("output.Count mismatch len(output.Books): got %d; expected %d", *resBody.Count, outLen)
				}

				if outLen != expLen {
					t.Fatalf("output had wrong len: got %d; expected %d", outLen, expLen)
				}

				for i, p := range pages {
					s := i * tc.countPerPage
					if s > tc.limit {
						break
					}
					e := clampMax(s+tc.countPerPage, tc.limit)
					if e > len(resBody.Books) {
						e = len(resBody.Books)
					}
					ep := clampMax(len(p), clampMax(tc.limit, int(defaultLimit)))
					if diff := cmp.Diff(p[:ep], resBody.Books[s:e], cmpopts.IgnoreFields(books.BestSellerBook{}, "Expiration")); diff != "" {
						t.Errorf("segment [%d:%d] of resBody.Books was not equal to input (page-%d)[:%d] (-want +got):\n%s", s, e, i, ep, diff)
					}
				}
			}
		})
	}
}

type dummyQueryAPIClient struct{}

func (d *dummyQueryAPIClient) Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return nil, nil
}

type stubDynamoDBQueryPaginatorAPI struct {
	np           int
	p            int
	limit        int
	pages        [][]books.BestSellerBook
	err          error
	pageErrorNum int
}

func (s *stubDynamoDBQueryPaginatorAPI) HasMorePages() bool {
	return s.p != s.np
}

func (s *stubDynamoDBQueryPaginatorAPI) NextPage(ctx context.Context, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if s.err != nil && s.p == s.pageErrorNum {
		return &dynamodb.QueryOutput{Count: 0}, s.err
	}

	t := s.p
	s.p++

	p := s.pages[t]
	items := []map[string]types.AttributeValue{}
	for _, v := range p {
		if s.limit == 0 {
			break
		}
		s.limit--
		item, _ := attributevalue.MarshalMap(v)
		items = append(items, item)
	}

	return &dynamodb.QueryOutput{Count: int32(len(p)), Items: items}, nil
}

type mockDynamoDBNewQueryPaginatorAPIProvider struct {
	*testing.T
	s *stubDynamoDBQueryPaginatorAPI
	q map[string]string
}

func (m *mockDynamoDBNewQueryPaginatorAPIProvider) mockDynamoDBNewScanPaginatorAPI(
	client dynamodb.QueryAPIClient, params *dynamodb.QueryInput, optFns ...func(*dynamodb.QueryPaginatorOptions),
) DynamoDBQueryPaginatorAPI {
	if params == nil {
		m.Fatalf("NewQueryPaginator: got nil params")
	}
	if *params.TableName != TableName {
		m.Errorf("NewQueryPaginator: got params.TableName with value %s; expected %s", *params.TableName, TableName)
	}

	if sLimit, ok := m.q["limit"]; ok {
		limit, _ := strconv.ParseInt(sLimit, 10, 32)
		if params.Limit == nil {
			m.Errorf("NewQueryPaginator: params.Limit not present in params: expected %d", limit)
		} else {
			pLimit := int64(*params.Limit)
			if pLimit != limit && pLimit != int64(defaultLimit) {
				m.Errorf("NewQueryPaginator: params.Limit not equal to params: got %d; expected %d", *params.Limit, limit)
			}
		}
	}
	if params.KeyConditionExpression == nil {
		m.Errorf("NewQueryPaginator: params.KeyConditionExpression was nil")
	}
	if params.ExpressionAttributeNames == nil {
		m.Errorf("NewQueryPaginator: params.ExpressionAttributeNames was nil")
	}
	if params.ExpressionAttributeValues == nil {
		m.Errorf("NewQueryPaginator: params.ExpressionAttributeValues was nil")
	}
	return m.s
}

const TableName = "Table"

func str(s string) *string {
	return &s
}

func containsError(errs []ErrorInfo, field string) int {
	for i, err := range errs {
		if err.Field == field {
			return i
		}
	}
	return -1
}

func describePStr(s *string) string {
	if s == nil {
		return "none"
	}
	return *s
}
