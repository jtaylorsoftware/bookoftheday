package books

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRequests(t *testing.T) {
	t.Run("401 unauthorized returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			fmt.Fprintf(w, "")
		}))
		defer ts.Close()

		api := &NYTBooksAPI{
			key:      "",
			endpoint: ts.URL,
		}

		if _, err := api.GetBestSellerListNames(); err == nil {
			t.Errorf("got nil error; expected non-nil")
		}
	})

	t.Run("200 unmarshals JSON successfully", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{
"status": "OK",
"copyright": "Copyright",
"num_results": 1,
"results": [
		{
			"list_name": "Fiction",
			"display_name": "Fiction",
			"list_name_encoded": "fiction",
			"oldest_published_date": "2022-06-14",
			"newest_published_date": "2022-06-14",
			"updated": "WEEKLY"
		}
	]
}
`)
		}))
		defer ts.Close()

		api := &NYTBooksAPI{
			key:      "",
			endpoint: ts.URL,
		}

		got, err := api.GetBestSellerListNames()
		if err != nil {
			t.Fatalf("got non-nil error; expected to succeed")
		}

		want := make([]BestSellerList, 1)
		want[0] = BestSellerList{
			Name:                "Fiction",
			DisplayName:         "Fiction",
			EncodedName:         "fiction",
			OldestPublishedDate: "2022-06-14",
			NewestPublishedDate: "2022-06-14",
			UpdatePeriod:        "WEEKLY",
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; expected %v", got, want)
		}
	})
}
