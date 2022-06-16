// Package api provides methods to access the NYT Books API and methods to
// persist Best-Seller lists to DynamoDB tables.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bookoftheday/types"
)

// NYTBooksAPI provides methods to query the NYT books API.
type NYTBooksAPI struct {
	key      string
	endpoint string
}

// NewNYTBooksAPI initializes a new instance of NYTBooksAPI.
func NewNYTBooksAPI(key, endpoint string) *NYTBooksAPI {
	return &NYTBooksAPI{key, endpoint}
}

type getListNamesResponse struct {
	Status     string                 `json:"status"`
	Copyright  string                 `json:"copyright"`
	NumResults int                    `json:"num_results"`
	Results    []types.BestSellerList `json:"results"`
}

// GetBestSellerListNames fetches the list of Best-Seller lists from the NYT
// books API.
func (api *NYTBooksAPI) GetBestSellerListNames() ([]types.BestSellerList, error) {
	resp, err := http.Get(
		api.endpoint + "?api-key=" + api.key,
	)
	if err != nil {
		return nil, fmt.Errorf("could not GET NYT Books API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response %d from Books API: %s", resp.StatusCode, string(body))
	}

	var data getListNamesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("could not unmarshal API response: %w", err)
	}

	return data.Results, nil
}
