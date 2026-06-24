package service

import (
	"testing"

	"dw0rdwk/backend/internal/models"
)

func TestDockingStatusFromOrderStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{status: "done", want: "sent"},
		{status: "processing", want: "sent"},
		{status: "failed", want: "failed"},
		{status: "cancelled", want: "cancelled"},
		{status: "refunded", want: "refunded"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := dockingStatusFromOrderStatus(tt.status); got != tt.want {
				t.Fatalf("dockingStatusFromOrderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWK29GeneratedPriceUsesCategoryRule(t *testing.T) {
	value := 2.5
	pricing := wk29PricingFromConnector(models.Connector{
		PriceMode:              "multiplier",
		PriceValue:             1.2,
		PriceRounding:          "none",
		CategoryPriceRulesJSON: `{"cat-1":{"priceMode":"fixed_add","priceValue":2.5,"priceRounding":"ceil"}}`,
	})

	if got := wk29GeneratedPrice(1.2, "cat-1", pricing); got != 4 {
		t.Fatalf("category price = %v, want 4", got)
	}
	if got := wk29GeneratedPrice(10, "cat-2", pricing); got != 12 {
		t.Fatalf("default price = %v, want 12", got)
	}
	if normalizeConnectorPriceValue(&value) != 2.5 {
		t.Fatal("normalizeConnectorPriceValue did not preserve explicit value")
	}
}

func TestAggregateWK29OrderSyncRowsCountsConnectorErrors(t *testing.T) {
	result := aggregateWK29OrderSyncRows([]WK29OrderSyncConnectorRow{
		{ConnectorID: 1, Fetched: 2, Matched: 1, Updated: 1, Skipped: 1},
		{ConnectorID: 2, Error: "upstream failed"},
	})

	if result.Connectors != 2 || result.Fetched != 2 || result.Matched != 1 || result.Updated != 1 || result.Skipped != 1 || result.Failed != 1 {
		t.Fatalf("unexpected aggregate: %+v", result)
	}
}
