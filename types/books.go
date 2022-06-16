// Package types provides common data types for querying and storing books.
package types

// BestSellerList models a single Best-Seller books list.
type BestSellerList struct {
	Name                string `json:"list_name"`
	DisplayName         string `json:"display_name"`
	EncodedName         string `json:"list_name_encoded"`
	OldestPublishedDate string `json:"oldest_published_date"`
	NewestPublishedDate string `json:"newest_published_date"`
	UpdatePeriod        string `json:"updated"`
}
