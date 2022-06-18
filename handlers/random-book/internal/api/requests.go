// Package api provides methods to make requests to the NYT Books API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// NYTBooksAPI provides methods to query the NYT books API for Best-Seller books.
type NYTBooksAPI struct {
	key string

	// A format string that takes 2 string arguments for the date and list name.
	endpoint string
}

// NewNYTBooksAPI initializes a new instance of NYTBooksAPI.
func NewNYTBooksAPI(key, endpoint string) *NYTBooksAPI {
	return &NYTBooksAPI{key, endpoint}
}

// GetBooksInListOnDate returns the best-selling books list for the best-seller list published on a given date.
// If the date doesn't exactly match a published date, the nearest in the future is returned.
func (api *NYTBooksAPI) GetBooksInListOnDate(list string, date string) (BestSellerBookList, error) {
	url := fmt.Sprintf(api.endpoint+"?api-key=%s", date, list, api.key)
	resp, err := http.Get(url)
	if err != nil {
		return BestSellerBookList{}, fmt.Errorf("could not GET NYT Books API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return BestSellerBookList{}, fmt.Errorf("error response %d from Books API: %s", resp.StatusCode, string(body))
	}

	var data getBooksInListOnDateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return BestSellerBookList{}, fmt.Errorf("could not unmarshal API response: %w", err)
	}

	return data.Results, nil
}
