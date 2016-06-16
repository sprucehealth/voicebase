package products

import (
	"bytes"
	"net/url"
	"os"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestScrape(t *testing.T) {
	u, err := url.Parse("https://example.com/123?blah=blah")
	test.OK(t, err)
	p, err := scrape(u, bytes.NewReader([]byte(`<!doctype html>
		<html>
		<head>
			<link rel="canonical" href="https://example.com/123">
			<meta property="og:title" content="This is a thing">
			<META PROPERTY="og:image" CONTENT="//example.com/image.jpeg">
		</head>
		<body>
		</body>
		</html>
	`)))
	test.OK(t, err)
	test.Equals(t, "url:https://example.com/123", p.ID)
	test.Equals(t, "https://example.com/123", p.ProductURL)
	test.Equals(t, "This is a thing", p.Name)
	test.Equals(t, 1, len(p.ImageURLs))
	test.Equals(t, "https://example.com/image.jpeg", p.ImageURLs[0])
}

func TestAmazonProducts(t *testing.T) {
	accessKey := os.Getenv("TEST_AMAZON_ACCESS_KEY")
	secretKey := os.Getenv("TEST_AMAZON_SECRET_KEY")
	associateTag := os.Getenv("TEST_AMAZON_AFFILIATE_TAG")
	if accessKey == "" || secretKey == "" || associateTag == "" {
		t.Skip("Missing TEST_AMAZON_ vars")
	}

	c, err := NewAmazonProductsClient(accessKey, secretKey, associateTag)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.LookupByASIN("B001V9SXXU")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", p)
}
