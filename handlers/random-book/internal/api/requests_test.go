package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRequests(t *testing.T) {
	t.Run("unmarshals API response successfully", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/date/list.json" {
				t.Errorf("request path invalid: got %s; expected %s", r.URL.Path, "/date/list.json")
			}
			data := `{
"status": "OK",
"copyright": "Copyright",
"num_results": 11,
"last_modified": "2022-06-09T12:34:56-07:00",
"results": {
	"list_name": "Hardcover",
	"list_name_encoded": "hardcover",
	"bestsellers_date": "2022-01-23",
	"published_date": "2022-03-21",
	"published_date_description": "one_before_latest",
	"next_published_date": "2021-01-11",
	"previous_published_date": "2021-02-22",
	"display_name": "Hardcover",
	"normal_list_ends_at": 11,
	"updated": "WEEKLY",
	"books": [
		{
			"rank": 1,
			"rank_last_week": 0,
			"weeks_on_list": 1,
			"asterisk": 0,
			"dagger": 0,
			"primary_isbn10": "1234567890",
			"primary_isbn13": "1234567890123",
			"publisher": "Test",
			"description": "TestBook",
			"price": "0.00",
			"title": "TEST-BOOK",
			"author": "Test",
			"contributor": "Test",
			"contributor_note": "ContributorNote",
			"book_image": "ImageLink",
			"book_image_width": 400,
			"book_image_height": 500,
			"amazon_product_url": "ProductURL",
			"age_group": "all",
			"book_review_link": "ReviewLink",
			"first_chapter_link": "ChapterLink",
			"sunday_review_link": "SundayReviewLink",
			"article_chapter_link": "ArticleChapterLink",
			"isbns": [
				{
					"isbn10": "1234567890",
					"isbn13": "1234567890123"
				}
			],
			"buy_links": [
				{
					"name": "Website1",
					"url": "URL1"
				}
			],
			"book_uri": "URI"
		}
	]
}
}
`
			fmt.Fprintf(w, data)
		}))

		defer ts.Close()

		api := &NYTBooksAPI{
			key:      "",
			endpoint: ts.URL + "/%s/%s.json",
		}

		got, err := api.GetBooksInListOnDate("list", "date")
		if err != nil {
			t.Fatalf("got err %v; expected nil", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("fields mismatch in unmarshalled response (-want +got):\n%s", diff)
		}
	})
}

var want = BestSellerBookList{
	ListName:                 "Hardcover",
	ListNameEncoded:          "hardcover",
	BestSellersDate:          "2022-01-23",
	PublishedDate:            "2022-03-21",
	PublishedDateDescription: "one_before_latest",
	NextPublishedDate:        "2021-01-11",
	PreviousPublishedDate:    "2021-02-22",
	DisplayName:              "Hardcover",
	NormalListEndsAt:         11,
	Updated:                  "WEEKLY",
	Books: []BestSellerBook{
		{
			Rank:               1,
			RankLastWeek:       0,
			WeeksOnList:        1,
			Asterisk:           0,
			Dagger:             0,
			PrimaryISBN10:      "1234567890",
			PrimaryISBN13:      "1234567890123",
			Publisher:          "Test",
			Description:        "TestBook",
			Price:              "0.00",
			Title:              "TEST-BOOK",
			Author:             "Test",
			Contributor:        "Test",
			ContributorNote:    "ContributorNote",
			ImageURL:           "ImageLink",
			ImageWidth:         400,
			ImageHeight:        500,
			AmazonProductURL:   "ProductURL",
			AgeGroup:           "all",
			ReviewLink:         "ReviewLink",
			FirstChapterLink:   "ChapterLink",
			SundayReviewLink:   "SundayReviewLink",
			ArticleChapterLink: "ArticleChapterLink",
			ISBNs: []ISBNPair{
				{
					ISBN10: "1234567890",
					ISBN13: "1234567890123",
				},
			},
			BuyLinks: []BuyLink{
				{
					Name: "Website1",
					URL:  "URL1",
				},
			},
			URI: "URI",
		},
	},
}
