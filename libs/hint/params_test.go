package hint

import "testing"

func TestListParams(t *testing.T) {
	t.Run("OnlyOffsetAndLimit", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitAndSort", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitSortAndOperations", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
			Items: []*QueryItem{
				{
					Field: "created_at",
					Operations: []*Operation{
						{
							Operator: OperatorGreaterThanEqualTo,
							Operand:  "2016-05-05",
						},
						{
							Operator: OperatorLessThanEqualTo,
							Operand:  "2016-12-05",
						},
					},
				},
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "created_at=%7B%22gte%22%3A%222016-05-05%22%2C%22lte%22%3A%222016-12-05%22}&sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})

	t.Run("OffsetLimitSortAndEqualTo", func(t *testing.T) {
		l := ListParams{
			Offset: 10,
			Limit:  100,
			Sort: &Sort{
				By:   "first_name",
				Desc: true,
			},
			Items: []*QueryItem{
				{
					Field: "created_at",
					Operations: []*Operation{
						{
							Operator: OperatorEqualTo,
							Operand:  "2016-05-05",
						},
					},
				},
			},
		}

		encoded, err := l.Encode()
		if err != nil {
			t.Fatal(err)
		}
		expected := "created_at=2016-05-05&sort=-first_name&offset=10&limit=100"
		if encoded != expected {
			t.Fatalf("Expected %s got %s", expected, encoded)
		}
	})
}
