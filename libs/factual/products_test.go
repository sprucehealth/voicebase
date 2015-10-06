package factual

import (
	"os"
	"testing"
)

func TestQueryProducts(t *testing.T) {
	key := os.Getenv("TEST_FACTUAL_KEY")
	secret := os.Getenv("TEST_FACTUAL_SECRET")
	if key == "" || secret == "" {
		t.Skip("Missing TEST_FACTUAL_KEY or TEST_FACTUAL_SECRET")
	}
	c := New(key, secret)
	ps, err := c.QueryProducts("shampoo", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%+v", p)
	}
}

func TestProduct(t *testing.T) {
	key := os.Getenv("TEST_FACTUAL_KEY")
	secret := os.Getenv("TEST_FACTUAL_SECRET")
	if key == "" || secret == "" {
		t.Skip("Missing TEST_FACTUAL_KEY or TEST_FACTUAL_SECRET")
	}

	c := New(key, secret)

	// Product exists
	p, err := c.Product("0f8334d1-acf8-4dc8-a64d-66a4450d1deb")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", p)

	// Product does not exist
	_, err = c.Product("0f8334d1-acf8-1234-1234-66a4450d1deb")
	if err != ErrNotFound {
		t.Fatalf("Expected ErrNotFound got %T: %s", err, err)
	}
}

func TestQueryProductsCrosswalk(t *testing.T) {
	key := os.Getenv("TEST_FACTUAL_KEY")
	secret := os.Getenv("TEST_FACTUAL_SECRET")
	if key == "" || secret == "" {
		t.Skip("Missing TEST_FACTUAL_KEY or TEST_FACTUAL_SECRET")
	}
	c := New(key, secret)

	ps, err := c.QueryProductsCrosswalk(map[string]*Filter{
		"factual_id": {
			In: []string{"288fc328-693e-4272-b062-118732f94048", "e9ee5603-4652-4ca4-be8b-8c467149025e", "51e9f746-cdaf-4a64-8b9b-cba3471068a4"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%+v", p)
	}

	ps, err = c.QueryProductsCrosswalk(map[string]*Filter{
		"factual_id": {
			Eq: "288fc328-693e-4272-b062-118732f94048",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%+v", p)
	}
}
