package connectivity

import (
	"context"
	"net/http"
	"testing"
)

// Every wrapper, pinned to the method and path it must call. These are thin
// by design, so the one thing that can be wrong is where they point — and a
// wrong path returns a plausible-looking zero value rather than an error.
func TestEveryRouteIsWhereItShouldBe(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		call   func(*Client) error
		method string
		path   string
	}{
		{"overview", func(c *Client) error { _, err := c.GetOverview(ctx); return err },
			"GET", "/v1/overview"},
		{"network overview", func(c *Client) error { _, err := c.GetNetworkOverview(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/overview"},
		{"audit", func(c *Client) error { _, err := c.GetAuditLog(ctx, AuditQuery{}); return err },
			"GET", "/v1/audit"},
		{"alerts", func(c *Client) error { _, err := c.NetworkAlerts(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/alerts"},
		{"application", func(c *Client) error { _, err := c.GetNetworkApplication(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/application"},

		{"create api key", func(c *Client) error { _, err := c.CreateApiKey(ctx, "n1", "k"); return err },
			"POST", "/v1/networks/n1/api-keys"},
		{"list api keys", func(c *Client) error { _, err := c.ListApiKeys(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/api-keys"},
		{"revoke api key", func(c *Client) error { return c.RevokeApiKey(ctx, "n1", "k1") },
			"DELETE", "/v1/networks/n1/api-keys/k1"},

		{"create integration", func(c *Client) error {
			_, err := c.CreateIntegration(ctx, "n1", IntegrationHTTP, nil)
			return err
		}, "POST", "/v1/networks/n1/integrations"},
		{"list integrations", func(c *Client) error { _, err := c.ListIntegrations(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/integrations"},
		{"update integration", func(c *Client) error {
			_, err := c.UpdateIntegration(ctx, "n1", "i1", UpdateIntegrationRequest{})
			return err
		}, "PUT", "/v1/networks/n1/integrations/i1"},
		{"delete integration", func(c *Client) error { return c.DeleteIntegration(ctx, "n1", "i1") },
			"DELETE", "/v1/networks/n1/integrations/i1"},
		{"test integration", func(c *Client) error { _, err := c.TestIntegration(ctx, "n1", "i1"); return err },
			"POST", "/v1/networks/n1/integrations/i1/test"},
		{"deliveries", func(c *Client) error { _, err := c.Deliveries(ctx, "n1", "i1", 0, 0); return err },
			"GET", "/v1/networks/n1/integrations/i1/deliveries"},
		{"retry delivery", func(c *Client) error { _, err := c.RetryDelivery(ctx, "n1", "i1", "d1"); return err },
			"POST", "/v1/networks/n1/integrations/i1/deliveries/d1/retry"},

		{"create event rule", func(c *Client) error {
			_, err := c.CreateEventRule(ctx, "n1", CreateEventRuleRequest{Name: "quiet", Kind: EventRuleQuiet})
			return err
		}, "POST", "/v1/networks/n1/event-rules"},
		{"list event rules", func(c *Client) error { _, err := c.ListEventRules(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/event-rules"},
		{"delete event rule", func(c *Client) error { return c.DeleteEventRule(ctx, "n1", "r1") },
			"DELETE", "/v1/networks/n1/event-rules/r1"},

		{"plans", func(c *Client) error { _, err := c.ListPlans(ctx); return err },
			"GET", "/v1/billing/plans"},
		{"subscription", func(c *Client) error { _, err := c.GetSubscription(ctx); return err },
			"GET", "/v1/billing/subscription"},
		{"invoices", func(c *Client) error { _, err := c.ListInvoices(ctx); return err },
			"GET", "/v1/billing/invoices"},
		{"payment methods", func(c *Client) error { _, err := c.ListCheckoutMethods(ctx); return err },
			"GET", "/v1/billing/payment-methods"},
		{"usage", func(c *Client) error { _, err := c.GetUsage(ctx); return err },
			"GET", "/v1/billing/usage"},

		{"list destinations", func(c *Client) error { _, err := c.ListDestinations(ctx); return err },
			"GET", "/v1/platforms"},
		{"create destination", func(c *Client) error {
			_, err := c.CreateDestination(ctx, CreateDestinationRequest{})
			return err
		}, "POST", "/v1/platforms"},
		{"update destination", func(c *Client) error {
			_, err := c.UpdateDestination(ctx, "p1", UpdateDestinationRequest{})
			return err
		}, "PUT", "/v1/platforms/p1"},
		{"delete destination", func(c *Client) error { return c.DeleteDestination(ctx, "p1") },
			"DELETE", "/v1/platforms/p1"},

		{"unclaimed", func(c *Client) error { _, err := c.ListUnclaimedDevices(ctx); return err },
			"GET", "/v1/devices/unclaimed"},
		{"staff billing", func(c *Client) error { _, err := c.StaffBillingSummary(ctx); return err },
			"GET", "/v1/admin/billing"},
		{"staff plans", func(c *Client) error { _, err := c.StaffListPlans(ctx); return err },
			"GET", "/v1/admin/plans"},
		{"staff create plan", func(c *Client) error { _, err := c.StaffCreatePlan(ctx, StaffPlanRequest{}); return err },
			"POST", "/v1/admin/plans"},
		{"staff update plan", func(c *Client) error {
			_, err := c.StaffUpdatePlan(ctx, "p1", StaffPlanRequest{})
			return err
		}, "PUT", "/v1/admin/plans/p1"},
		{"staff delete plan", func(c *Client) error { return c.StaffDeletePlan(ctx, "p1") },
			"DELETE", "/v1/admin/plans/p1"},
		{"staff assign plan", func(c *Client) error { _, err := c.StaffAssignPlan(ctx, "b1", "p1"); return err },
			"PUT", "/v1/admin/businesses/b1/plan"},

		{"device metrics", func(c *Client) error { _, err := c.GetDeviceMetrics(ctx, "d1", "24h"); return err },
			"GET", "/v1/devices/d1/metrics"},
		{"device activity", func(c *Client) error { _, err := c.GetDeviceActivity(ctx, "d1", 0, 0); return err },
			"GET", "/v1/devices/d1/activity"},
		{"device logs", func(c *Client) error { _, err := c.DeviceLogs(ctx, "d1", FrameQuery{}); return err },
			"GET", "/v1/devices/d1/logs"},
		{"access points", func(c *Client) error { _, err := c.ListAccessPoints(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/access-points"},
		{"device classes", func(c *Client) error { _, err := c.ListDeviceClasses(ctx, "n1"); return err },
			"GET", "/v1/networks/n1/device-classes"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				writeEnvelope(w, 200, nil)
			})
			if err := tc.call(client); err != nil {
				t.Fatalf("call: %v", err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Errorf("%s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
		})
	}
}

// The platform API is reached with a network API key and lives under its own
// prefix; a method that drifts onto the console path would authenticate
// differently and silently address another scope.
func TestPlatformRoutesKeepTheirPrefix(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		call   func(*PlatformClient) error
		method string
		path   string
	}{
		{"network", func(p *PlatformClient) error { _, err := p.Network(ctx); return err },
			"GET", "/v1/platform/network"},
		{"device classes", func(p *PlatformClient) error { _, err := p.DeviceClasses(ctx); return err },
			"GET", "/v1/platform/device-classes"},
		{"create", func(p *PlatformClient) error {
			_, err := p.CreateDevice(ctx, CreatePlatformDeviceRequest{})
			return err
		}, "POST", "/v1/platform/devices"},
		{"get", func(p *PlatformClient) error { _, err := p.GetDevice(ctx, "d1"); return err },
			"GET", "/v1/platform/devices/d1"},
		{"update", func(p *PlatformClient) error {
			_, err := p.UpdateDevice(ctx, "d1", UpdatePlatformDeviceRequest{})
			return err
		}, "PUT", "/v1/platform/devices/d1"},
		{"list", func(p *PlatformClient) error { _, err := p.ListDevices(ctx, 0, 0); return err },
			"GET", "/v1/platform/devices"},
		{"delete", func(p *PlatformClient) error { return p.DeleteDevice(ctx, "d1") },
			"DELETE", "/v1/platform/devices/d1"},
		{"set key", func(p *PlatformClient) error { _, err := p.SetKey(ctx, "d1", "k"); return err },
			"PUT", "/v1/platform/devices/d1/key"},
		{"clear key", func(p *PlatformClient) error { _, err := p.ClearKey(ctx, "d1"); return err },
			"DELETE", "/v1/platform/devices/d1/key"},
		{"enable", func(p *PlatformClient) error { _, err := p.Enable(ctx, "d1"); return err },
			"POST", "/v1/platform/devices/d1/enable"},
		{"disable", func(p *PlatformClient) error { _, err := p.Disable(ctx, "d1"); return err },
			"POST", "/v1/platform/devices/d1/disable"},
		{"queue command", func(p *PlatformClient) error {
			_, err := p.QueueCommand(ctx, "d1", []byte{1}, CommandOptions{})
			return err
		}, "POST", "/v1/platform/devices/d1/commands"},
		{"pending commands", func(p *PlatformClient) error { _, err := p.PendingCommands(ctx, "d1"); return err },
			"GET", "/v1/platform/devices/d1/commands"},
		{"clear commands", func(p *PlatformClient) error { return p.ClearCommands(ctx, "d1") },
			"DELETE", "/v1/platform/devices/d1/commands"},
		{"frames", func(p *PlatformClient) error { _, err := p.Frames(ctx, "d1", FrameQuery{}); return err },
			"GET", "/v1/platform/devices/d1/frames"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			srv := newRecordingServer(t, &gotMethod, &gotPath, &gotAuth)
			p := NewPlatform(srv.URL+"/v1", "api-key-token")
			if err := tc.call(p); err != nil {
				t.Fatalf("call: %v", err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Errorf("%s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
			if gotAuth != "Bearer api-key-token" {
				t.Errorf("the network API key was not sent: %q", gotAuth)
			}
		})
	}
}
