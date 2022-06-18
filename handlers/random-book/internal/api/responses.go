package api

type getBooksInListOnDateResponse struct {
	Status       string             `json:"status"`
	Copyright    string             `json:"copyright"`
	NumResults   int                `json:"num_results"`
	LastModified string             `json:"last_modified"`
	Results      BestSellerBookList `json:"results"`
}

type BestSellerBookList struct {
	ListName                 string           `json:"list_name"`
	ListNameEncoded          string           `json:"list_name_encoded"`
	BestSellersDate          string           `json:"bestsellers_date"`
	PublishedDate            string           `json:"published_date"`
	PublishedDateDescription string           `json:"published_date_description"`
	NextPublishedDate        string           `json:"next_published_date"`
	PreviousPublishedDate    string           `json:"previous_published_date"`
	DisplayName              string           `json:"display_name"`
	NormalListEndsAt         int              `json:"normal_list_ends_at"`
	Updated                  string           `json:"updated"`
	Books                    []BestSellerBook `json:"books"`
}

type BestSellerBook struct {
	Rank               int        `json:"rank"`
	RankLastWeek       int        `json:"rank_last_week"`
	WeeksOnList        int        `json:"weeks_on_list"`
	Asterisk           int        `json:"asterisk"`
	Dagger             int        `json:"dagger"`
	PrimaryISBN10      string     `json:"primary_isbn10"`
	PrimaryISBN13      string     `json:"primary_isbn13"`
	Publisher          string     `json:"publisher"`
	Description        string     `json:"description"`
	Price              string     `json:"price"`
	Title              string     `json:"title"`
	Author             string     `json:"author"`
	Contributor        string     `json:"contributor"`
	ContributorNote    string     `json:"contributor_note"`
	ImageURL           string     `json:"book_image"`
	ImageWidth         int        `json:"book_image_width"`
	ImageHeight        int        `json:"book_image_height"`
	AmazonProductURL   string     `json:"amazon_product_url"`
	AgeGroup           string     `json:"age_group"`
	ReviewLink         string     `json:"book_review_link"`
	FirstChapterLink   string     `json:"first_chapter_link"`
	SundayReviewLink   string     `json:"sunday_review_link"`
	ArticleChapterLink string     `json:"article_chapter_link"`
	ISBNs              []ISBNPair `json:"isbns"`
	BuyLinks           []BuyLink  `json:"buy_links"`
	URI                string     `json:"book_uri"`
}

type ISBNPair struct {
	ISBN10 string `json:"isbn10"`
	ISBN13 string `json:"isbn13"`
}

type BuyLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
