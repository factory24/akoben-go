package connectivity

import (
	"context"
	"net/http"
	"testing"
)

// The token is minted once and only hashed afterwards. A caller that does not
// read it here can never recover it, so it must survive decoding intact.
func TestCreateApiKeyReturnsTheTokenOnce(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeEnvelope(w, 201, map[string]any{
				"id": "k-1", "networkId": "n-1", "name": "flow",
				"prefix": "9f2a10cc", "token": "9f2a10cc" + "00112233445566778899aabbccddeeff00112233",
				"createdAt": 1789500000,
			})
			return
		}
		writeEnvelope(w, 200, []map[string]any{
			{"id": "k-1", "networkId": "n-1", "name": "flow", "prefix": "9f2a10cc", "createdAt": 1789500000},
		})
	})
	key, err := client.CreateApiKey(context.Background(), "n-1", "flow")
	if err != nil {
		t.Fatal(err)
	}
	if len(key.Token) != 48 {
		t.Errorf("token = %q (%d chars), want the 48-char token", key.Token, len(key.Token))
	}
	keys, err := client.ListApiKeys(context.Background(), "n-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Token != "" {
		t.Errorf("a listing handed back key material: %+v", keys)
	}
}

// A refused delivery is a successful call. A client that treats err == nil as
// "it worked" reports a broken destination as healthy.
func TestFailedTestDeliveryIsNotAnError(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{
			"kind": "http", "success": false, "statusCode": 502,
			"latencyMs": 231, "error": "bad gateway",
		})
	})
	res, err := client.TestIntegration(context.Background(), "n-1", "i-1")
	if err != nil {
		t.Fatalf("err = %v, want nil — the call succeeded, the delivery did not", err)
	}
	if res.Success {
		t.Error("success = true on a 502 delivery")
	}
	if res.StatusCode != 502 || res.Error != "bad gateway" {
		t.Errorf("res = %+v", res)
	}
}

// A queue destination has no HTTP status at all; absent must not read as 0.
func TestTestDeliveryWithoutAnHTTPStatus(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, 200, map[string]any{"kind": "pulsar", "success": true, "latencyMs": 12})
	})
	res, err := client.TestIntegration(context.Background(), "n-1", "i-1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.StatusCode != 0 {
		t.Errorf("res = %+v", res)
	}
}

func TestDeliveriesArePagedAndCarryRetryable(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "page=2&pageSize=50" {
			t.Errorf("query = %q", got)
		}
		writeEnvelope(w, 200, map[string]any{
			"content": []map[string]any{
				{"id": "d-1", "integrationId": "i-1", "at": 1789500000, "success": false,
					"httpStatus": 500, "durationMs": 91, "retryable": true},
				{"id": "d-2", "integrationId": "i-1", "at": 1789400000, "success": true,
					"durationMs": 40, "retryable": false},
			},
			"totalElements": 2, "page": 2, "pageSize": 50,
		})
	})
	page, err := client.Deliveries(context.Background(), "n-1", "i-1", 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalElements != 2 || len(page.Content) != 2 {
		t.Fatalf("page = %+v", page)
	}
	if !page.Content[0].Retryable || page.Content[1].Retryable {
		t.Error("retryable did not survive — it is what gates the retry")
	}
	if page.Content[0].HTTPStatus == nil || *page.Content[0].HTTPStatus != 500 {
		t.Errorf("httpStatus = %v", page.Content[0].HTTPStatus)
	}
	if page.Content[1].HTTPStatus != nil {
		t.Error("an absent httpStatus decoded as a number")
	}
}

// Retrying a delivery with no stored payload is refused with a 400, not a 200
// carrying success false — the two failure shapes are different.
func TestRetryWithoutAStoredPayloadIsARefusal(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeErrors(w, http.StatusBadRequest,
			"bad_request: this delivery has no stored payload to retry")
	})
	_, err := client.RetryDelivery(context.Background(), "n-1", "i-1", "d-9")
	if err == nil {
		t.Fatal("a delivery with no payload was retried")
	}
	if IsNotFound(err) || IsConflict(err) {
		t.Errorf("err classified as not-found/conflict: %v", err)
	}
}

func TestIntegrationConfigHelpers(t *testing.T) {
	h := HTTPConfig{URL: "https://example.test/hook", AuthHeader: "X-Auth", AuthSecret: "s3cret"}.Config()
	if h["url"] != "https://example.test/hook" || h["authHeader"] != "X-Auth" || h["authSecret"] != "s3cret" {
		t.Errorf("http config = %v", h)
	}
	if bare := (HTTPConfig{URL: "https://example.test/hook"}).Config(); len(bare) != 1 {
		t.Errorf("an unauthenticated destination sent empty auth keys: %v", bare)
	}
	q := QueueConfig{BrokerURL: "pulsar://broker:6650", Topic: "readings"}.Config()
	if q["brokerUrl"] != "pulsar://broker:6650" || q["topic"] != "readings" {
		t.Errorf("queue config = %v", q)
	}
}
