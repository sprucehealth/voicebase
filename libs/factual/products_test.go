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
	ps, err := c.QueryProducts("shampoo")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		t.Logf("%+v", p)
	}
}
