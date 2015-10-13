package products

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/dominicphillips/amazing"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/products"
	"golang.org/x/net/html"
)

func scrape(u *url.URL, r io.Reader) (*products.Product, error) {
	page, err := parseHTML(r)
	if err != nil {
		return nil, err
	}
	p := &products.Product{
		ProductURL: u.String(),
	}
	if earl, ok := normalizeURL(u, page.meta["og:url"]); ok {
		p.ProductURL = earl
	} else if earl, ok := normalizeURL(u, page.canonicalURL); ok {
		p.ProductURL = earl
	}
	if t := page.meta["og:title"]; t != "" {
		p.Name = t
	} else if t := page.meta["twitter:title"]; t != "" {
		p.Name = t
	} else if t := page.meta["title"]; t != "" {
		p.Name = t
	} else {
		p.Name = strings.TrimSpace(string(page.title))
	}
	if u.Host == "www.dermalogica.com" {
		// They have a meta for the image but it's tiny so grab the large iamge by class. Fall back to meta if this fails.
		if earl, ok := normalizeURL(u, page.imgByClass["main-product-image"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		}
	} else if u.Host == "www.narscosmetics.com" {
		// They don't provide any meta tags for the image so need to get it from the a tag (img tag is not easy to query as it has no classes).
		if earl, ok := normalizeURL(u, page.linkByClass["main-image"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		}
	}
	if len(p.ImageURLs) == 0 {
		if earl, ok := normalizeURL(u, page.meta["og:image"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		} else if earl, ok := normalizeURL(u, page.meta["twitter:image"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		} else if earl, ok := normalizeURL(u, page.meta["twitter:image:src"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		} else if earl, ok := normalizeURL(u, page.meta["image"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		} else if earl, ok := normalizeURL(u, page.schemaImg); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		} else if earl, ok := normalizeURL(u, page.imgByClass["product-img"]); ok {
			p.ImageURLs = append(p.ImageURLs, earl)
		}
	}
	p.ID = "url:" + p.ProductURL
	return p, nil
}

// normalizeURL does basic URL validation and returns a normalzied (absolute) URL
func normalizeURL(base *url.URL, earl string) (string, bool) {
	if earl == "" {
		return "", false
	}
	// Make relative URLs absolute
	if strings.HasPrefix(earl, "//") { // scheme relative
		earl = base.Scheme + ":" + earl
	} else if earl[0] == '/' { // path relative
		earl = base.Scheme + "://" + base.Host + earl
	}
	u, err := url.Parse(earl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	return u.String(), true
}

var (
	bA        = []byte("a")
	bClass    = []byte("class")
	bContent  = []byte("content")
	bHead     = []byte("head")
	bHref     = []byte("href")
	bImg      = []byte("img")
	bItemProp = []byte("itemprop")
	bLink     = []byte("link")
	bMeta     = []byte("meta")
	bName     = []byte("name")
	bProperty = []byte("property")
	bRel      = []byte("rel")
	bSrc      = []byte("src")
	bTitle    = []byte("title")
)

type page struct {
	canonicalURL string
	schemaImg    string
	title        []byte
	meta         map[string]string
	imgByClass   map[string]string
	linkByClass  map[string]string
}

func parseHTML(r io.Reader) (*page, error) {
	p := &page{
		meta:        make(map[string]string),
		imgByClass:  make(map[string]string),
		linkByClass: make(map[string]string),
	}
	z := html.NewTokenizer(r)

	title := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			e := z.Err()
			if e == io.EOF {
				return p, nil
			}
			return p, z.Err()
		case html.TextToken:
			if title {
				p.title = append(p.title, z.Text()...)
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			switch {
			case bytes.Equal(tn, bTitle):
				title = true
			case bytes.Equal(tn, bMeta):
				var k, v []byte
				var prop string
				var content string
				for hasAttr {
					k, v, hasAttr = z.TagAttr()
					switch {
					case bytes.Equal(k, bContent):
						content = strings.TrimSpace(string(v))
					case bytes.Equal(k, bProperty) || bytes.Equal(k, bItemProp) || bytes.Equal(k, bName):
						prop = strings.TrimSpace(string(v))
					}
				}
				if prop != "" && content != "" {
					p.meta[prop] = content
				}
			case bytes.Equal(tn, bLink):
				var k, v []byte
				var rel string
				var href string
				for hasAttr {
					k, v, hasAttr = z.TagAttr()
					switch {
					case bytes.Equal(k, bRel):
						rel = strings.TrimSpace(string(v))
					case bytes.Equal(k, bHref):
						href = strings.TrimSpace(string(v))
					}
				}
				switch rel {
				case "canonical":
					p.canonicalURL = href
				}
			case bytes.Equal(tn, bImg):
				var k, v []byte
				var src string
				var itemProp string
				var classes []string
				for hasAttr {
					k, v, hasAttr = z.TagAttr()
					switch {
					case bytes.Equal(k, bSrc):
						src = strings.TrimSpace(string(v))
					case bytes.Equal(k, bItemProp):
						itemProp = strings.TrimSpace(string(v))
					case bytes.Equal(k, bClass):
						for _, c := range strings.Split(string(v), " ") {
							c = strings.TrimSpace(c)
							if c != "" {
								classes = append(classes, c)
							}
						}
					}
				}
				if src != "" {
					if itemProp == "image" {
						p.schemaImg = src
					} else {
						for _, c := range classes {
							p.imgByClass[c] = src
						}
					}
				}
			case bytes.Equal(tn, bA):
				var k, v []byte
				var href string
				var classes []string
				for hasAttr {
					k, v, hasAttr = z.TagAttr()
					switch {
					case bytes.Equal(k, bHref):
						href = strings.TrimSpace(string(v))
					case bytes.Equal(k, bClass):
						for _, c := range strings.Split(string(v), " ") {
							c = strings.TrimSpace(c)
							if c != "" {
								classes = append(classes, c)
							}
						}
					}
				}
				if href != "" {
					for _, c := range classes {
						p.linkByClass[c] = href
					}
				}
			}
		case html.EndTagToken:
			// Only parse the head
			tn, _ := z.TagName()
			switch {
			case bytes.Equal(tn, bTitle):
				title = false
			}
		}
	}
}

// NewAmazonProductsClient returns a new client to access the amazon associates ad products API.
func NewAmazonProductsClient(accessKey, secretKey, associateTag string) (AmazonProductClient, error) {
	cli, err := amazing.NewAmazing("US", associateTag, accessKey, secretKey)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return azc{c: cli}, nil
}

type azc struct {
	c *amazing.Amazing
}

func (az azc) LookupByASIN(asin string) (*products.Product, error) {
	res, err := az.c.ItemLookupAsin(asin, url.Values{
		"ResponseGroup": []string{"ItemAttributes,Images,VariationImages"},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !res.AmazonItems.Request.IsValid {
		return nil, errors.Trace(fmt.Errorf("products: invalid amazon API request: %+v", res.AmazonItems.Request.Errors))
	}
	if len(res.AmazonItems.Items) == 0 {
		return nil, products.ErrNotFound
	}
	// Only use the first item as there should really be only one
	it := res.AmazonItems.Items[0]
	p := &products.Product{
		ID:         "amazon:" + asin,
		ProductURL: it.DetailPageURL,
		Name:       it.ItemAttributes.Title,
	}
	p.ImageURLs = make([]string, 0, 1+len(it.ImageSets))
	if it.LargeImage.URL != "" {
		p.ImageURLs = append(p.ImageURLs, it.LargeImage.URL)
	}
	for _, is := range it.ImageSets {
		// One of the images in the set normally matches the main image
		if is.LargeImage.URL != "" && is.LargeImage.URL != it.LargeImage.URL {
			p.ImageURLs = append(p.ImageURLs, is.LargeImage.URL)
		}
	}
	return p, nil
}
