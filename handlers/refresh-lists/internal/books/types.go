// Package books provides methods to access the NYT Books API and methods to
// persist Best-Seller lists to DynamoDB tables.
package books

// BestSellerList models a single Best-Seller books list.
type BestSellerList struct {
	ListName            string `json:"list_name"`
	DisplayName         string `json:"display_name"`
	ListNameEncoded     string `json:"list_name_encoded"`
	OldestPublishedDate string `json:"oldest_published_date"`
	NewestPublishedDate string `json:"newest_published_date"`
	Updated             string `json:"updated"`
}
