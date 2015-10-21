package yelp

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Review struct {
	Excerpt             string  `json:"excerpt"`
	ID                  string  `json:"id"`
	Rating              float64 `json:"rating"`
	RatingImageSmallURL string  `json:"rating_image_small_url"`
	RatingImageLargeURL string  `json:"rating_image_large_url"`
	TimeCreated         int64   `json:"time_created"`
	User                *User   `json:"user"`
}

type Business struct {
	// ID is the Yelp ID for the business
	ID string `json:"id"`

	//IsClaimed indicates whether business has been claimed by owner
	IsClaimed bool `json:"is_claimed"`

	// IsClosed indicates whether business is permanently closed
	IsClosed bool `json:"is_closed"`

	// Name indicates name of business
	Name string `json:"name"`

	// ImageURL indicates url of photo for the business
	ImageURL string `json:"image_url"`

	// URL is the URL for the business page on Yelp
	URL string `json:"url"`

	// MobileURL is the URL for the mobile business page
	MobileURL string `json:"mobile_url"`

	// Phone is the phone number of the business with international dialing code
	Phone string `json:"phone"`

	// DisplayPhone is the phone number of the business formatted for display
	DisplayPhone string `json:"display_phone"`

	// ReviewCount represents number of reviews for the business
	ReviewCount int `json:"review_count"`

	// Rating represents the rating for this business
	Rating float64 `json:"rating"`

	// Contains reviews associated with the business
	Reviews []*Review `json:"reviews"`

	//LargeRatingImgURL is URL for large version of rating image for this business (size = 166x30)
	LargeRatingImgURL string `json:"rating_img_url_large"`

	// ... and more (not including for now as list is long and all fields not needed. Rest of fields
	// can be found at https://www.yelp.com/developers/documentation/v2/business)
}

type Client interface {
	Business(businessID string) (*Business, error)
}

type client struct {
	oauthClient oauth.Client
	token       oauth.Credentials
}

type yelpError struct {
	E struct {
		Text  string `json:"text"`
		ID    string `json:"id"`
		Field string `json:"field"`
	} `json:"error"`
}

func (y *yelpError) Error() string {
	return fmt.Sprintf("yelp error:\nText:%s\nID:%s\nField:%s\n", y.E.Text, y.E.ID, y.E.Field)
}

func (c *client) get(urlStr string, params url.Values, v interface{}) error {
	resp, err := c.oauthClient.Get(nil, &c.token, urlStr, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var ye yelpError
		if err := json.NewDecoder(resp.Body).Decode(&ye); err == nil {
			return &ye
		}
		return fmt.Errorf("yelp status %d ", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func NewClient(consumerKey, consumerSecret, token, secret string) Client {
	c := &client{
		token: oauth.Credentials{
			Token:  token,
			Secret: secret,
		},
	}
	c.oauthClient.Credentials = oauth.Credentials{
		Token:  consumerKey,
		Secret: consumerSecret,
	}
	return c
}

func (c *client) Business(businessID string) (*Business, error) {
	var b Business
	if err := c.get("http://api.yelp.com/v2/business/"+businessID, url.Values{}, &b); err != nil {
		return nil, err
	}

	return &b, nil
}
