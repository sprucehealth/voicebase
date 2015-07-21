package cost

import (
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/config"
)

type mockDataAPITotalCost struct {
	api.DataAPI
	itemCost      *common.ItemCost
	visits        []*common.PatientVisit
	accountCredit common.AccountCredit
}

func (m *mockDataAPITotalCost) GetActiveItemCost(skuType string) (*common.ItemCost, error) {
	return m.itemCost, nil
}
func (m *mockDataAPITotalCost) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return 0, nil
}
func (m *mockDataAPITotalCost) VisitsSubmittedForPatientSince(patientID int64, since time.Time) ([]*common.PatientVisit, error) {
	return m.visits, nil
}
func (m *mockDataAPITotalCost) PendingPromotionsForAccount(id int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error) {
	return nil, nil
}
func (m *mockDataAPITotalCost) AccountCredit(id int64) (*common.AccountCredit, error) {
	return &m.accountCredit, nil
}

// TestTotalCost_NoLaunchPromo ensures that querying for cost without a launch promo works as expected
func TestTotalCost_NoLaunchPromo(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeDisabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, false, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 4000 {
		t.Fatalf("Expected cost to be 0 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 1 {
		t.Fatalf("Expected %d line items but got %d", 1, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	}
}

// TestLaunchPromo_NoVisitsSubmitted ensures that querying for cost when launch promo is running and no visit has been submitted
// by patient since the launch promo start gives the visit for free to patient
func TestLaunchPromo_NoVisitsSubmitted(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeEnabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, false, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 0 {
		t.Fatalf("Expected cost to be 0 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 2 {
		t.Fatalf("Expected %d line items but got %d", 2, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	} else if costBreakdown.LineItems[1].Description != "Free Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Free Visit", costBreakdown.LineItems[1].Description)
	}
}

// TestLaunchPromo_VisitSubmittedButNotCharged ensures that querying for cost with the intent to commit the cost
// for a visit that was just submitted during launch promo is indeed free.
func TestLaunchPromo_VisitSubmittedButNotCharged(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
		visits: []*common.PatientVisit{
			{
				Status: common.PVStatusSubmitted,
			},
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeEnabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, true, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 0 {
		t.Fatalf("Expected cost to be 0 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 2 {
		t.Fatalf("Expected %d line items but got %d", 2, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	} else if costBreakdown.LineItems[1].Description != "Free Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Free Visit", costBreakdown.LineItems[1].Description)
	}
}

// TestLaunchPromo_FirstVisit_AccountCredit ensures that no account credit is used
// for first visit during launch promo.
func TestLaunchPromo_FirstVisit_AccountCredit(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
		accountCredit: common.AccountCredit{
			Credit: 1000,
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeEnabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, true, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 0 {
		t.Fatalf("Expected cost to be 0 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 2 {
		t.Fatalf("Expected %d line items but got %d", 2, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	} else if costBreakdown.LineItems[1].Description != "Free Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Free Visit", costBreakdown.LineItems[1].Description)
	}
}

// TestLaunchPromo_SecondVisitOnwards ensures that during the launch promo all visits beyond the first
// are charged for.
func TestLaunchPromo_SecondVisitOnwards(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
		visits: []*common.PatientVisit{
			{
				Status: common.PVStatusSubmitted,
			},
			{
				Status: common.PVStatusTreated,
			},
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeEnabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, false, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 4000 {
		t.Fatalf("Expected cost to be 4000 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 1 {
		t.Fatalf("Expected %d line items but got %d", 1, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	}
}

// TestLaunchPromo_SecondVisitOnward_OnCharge ensures that when attempting to commit cost
// all visits beyond the first one are charged for.
func TestLaunchPromo_SecondVisitOnward_OnCharge(t *testing.T) {
	m := &mockDataAPITotalCost{
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Description: "Spruce Visit",
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
		visits: []*common.PatientVisit{
			{Status: common.PVStatusSubmitted},
			{Status: common.PVStatusTreated},
		},
	}

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.GlobalFirstVisitFreeEnabled})
	test.OK(t, err)

	costBreakdown, err := totalCostForItems([]string{"test"}, 0, true, m, nil, cfgStore)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if costBreakdown.TotalCost.Amount != 4000 {
		t.Fatalf("Expected cost to be 4000 but it was %d", costBreakdown.TotalCost.Amount)
	} else if len(costBreakdown.LineItems) != 1 {
		t.Fatalf("Expected %d line items but got %d", 1, len(costBreakdown.LineItems))
	} else if costBreakdown.LineItems[0].Description != "Spruce Visit" {
		t.Fatalf("Expected line item '%s' but got '%s'", "Spruce Visit", costBreakdown.LineItems[0].Description)
	}

}
