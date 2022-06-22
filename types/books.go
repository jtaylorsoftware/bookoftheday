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

// BestSellerBook models a single Best-Seller book.
type BestSellerBook struct {
	ListEncodedName   string `json:"list_encoded_name"`
	DateSelected      string `json:"date_selected"`
	ListPublishedDate string `json:"list_published_date"`
	ListDisplayName   string `json:"list_display_name"`
	ListUpdatePeriod  string `json:"list_update_period"`
	PrimaryISBN10     string `json:"primary_isbn10"`
	PrimaryISBN13     string `json:"primary_isbn13"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	Publisher         string `json:"publisher"`
	Description       string `json:"description"`
	Rank              int    `json:"rank"`
	AmazonProductURL  string `json:"amazon_product_url"`
	ImageURL          string `json:"image_url"`
	ImageWidth        int    `json:"image_width"`
	ImageHeight       int    `json:"image_height"`
	Expiration        int64  `json:"-"`
}

// BookItemKey contains the primary key data for a BestSellerBook item.
type BookItemKey struct {
	ListEncodedName string `json:"list_encoded_name"`
	DateSelected    string `json:"date_selected"`
}

// SQSBookMessageBody models the SQS MessageBody data sent by the
// contacts Lambda for consumption by the send-email Lambda. The
// send-email Lambda can unmarshal records into this value.
type SQSBookMessageBody struct {
	ContactEmail string         `json:"contact_email"`
	Book         BestSellerBook `json:"book"`
}
