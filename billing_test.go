package connectivity

import (
	"context"
	"net/http"
	"testing"
)

// Quantities and money are decimal strings on this API. Decoding a quantity
// into an integer fails outright, and rounding money in a client is worse.
func TestUsageQuantityAndMoneyStayStrings(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"periodStart": 1789500000, "periodEnd": 1789600000,
			"byMetric": map[string]any{
				"messages":   map[string]any{"quantity": "1204", "amount": "12.04"},
				"cellularMb": map[string]any{"quantity": "18.5"},
			},
			"estimatedTotal": "12.04", "currency": "GHS", "unrated": []string{"commands"},
		})
	})
	u, err := client.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if got := u.ByMetric["messages"].Quantity; got != "1204" {
		t.Errorf("quantity = %q", got)
	}
	if got := u.ByMetric["cellularMb"].Quantity; got != "18.5" {
		t.Errorf("a fractional quantity did not survive: %q", got)
	}
	if u.Currency != "GHS" || u.EstimatedTotal != "12.04" {
		t.Errorf("currency=%q total=%q", u.Currency, u.EstimatedTotal)
	}
	if len(u.Unrated) != 1 {
		t.Errorf("unrated = %v", u.Unrated)
	}
}

// Having no subscription is an answer, not a failure: the API sends data null.
func TestNoSubscriptionIsNilNotAnError(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, nil)
	})
	sub, err := client.GetSubscription(context.Background())
	if err != nil {
		t.Fatalf("a business with no subscription errored: %v", err)
	}
	if sub != nil {
		t.Errorf("subscription = %+v, want nil", sub)
	}
}

func TestSubscriptionCarriesItsPlan(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"id": "sub-1", "status": "active", "startedAt": 1789500000,
			"plan": map[string]any{"id": "p-1", "code": "growth", "priceMonthly": "450.00", "currency": "NAD"},
		})
	})
	sub, err := client.GetSubscription(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sub == nil || sub.Plan == nil {
		t.Fatalf("subscription = %+v", sub)
	}
	if sub.Plan.Currency != "NAD" || sub.Plan.PriceMonthly != "450.00" {
		t.Errorf("plan = %+v", sub.Plan)
	}
}
