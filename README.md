# akoben-go

`github.com/factory24/akoben-go` — the Go client for the Akoben Networks device-connectivity
platform. Standard library only: no protobuf, no gRPC, no transitive dependencies. Every type is
owned here, and nothing names the technology delivering the connectivity.

The package is `connectivity`:

```go
import connectivity "github.com/factory24/akoben-go"
```

## Install

```
go get github.com/factory24/akoben-go
```

## Two clients, two credentials

| | `Client` | `PlatformClient` |
|---|---|---|
| Credential | an operator token (OIDC) | one network's API key |
| Scope | every network in the business | the one network the key is bound to |
| For | consoles, internal tooling | a customer's own backend |

A network API key is not a device key. The key on a device gates that device's traffic and is set
through `SetKey`; the network API key authenticates your service to us.

## Client — operator token

```go
// Explicit
client := connectivity.New("https://api.example.com/v1",
    connectivity.NewClientCredentialsTokenSource(tokenURL, clientID, clientSecret))

// Or from the environment
client, err := connectivity.NewFromEnv()
```

| Variable | Meaning |
|---|---|
| `CONNECTIVITY.BASE_URL` | API root including `/v1` (required) |
| `CONNECTIVITY.API_TOKEN` | static bearer token, **or** the three below |
| `CONNECTIVITY.TOKEN_URL` | OpenID token endpoint (client-credentials grant) |
| `CONNECTIVITY.CLIENT_ID` / `CONNECTIVITY.CLIENT_SECRET` | the service account |
| `CONNECTIVITY.BUSINESS_ID` | only for staff principals acting for one business |

Underscored spellings (`CONNECTIVITY_BASE_URL`) work too. `connectivity.NewDisabled(reason)`
returns a client whose every call fails with that reason — wire it unconditionally and let
unconfigured environments degrade per call instead of failing at boot.

```go
dev, err := client.RegisterDevice(ctx, connectivity.RegisterDeviceRequest{
    NetworkID: networkID, DeviceID: "0102030405060708",
    Name: "Meter 4", DeviceClassID: classID,
})

// Credentials are write-only. There is no read-back, by design.
err = client.SetDeviceSecret(ctx, dev.DeviceID, "00112233445566778899aabbccddeeff")

// Commands carry opaque bytes; the platform never interprets them.
commandID, err := client.SendCommand(ctx, dev.DeviceID, payload,
    connectivity.CommandOptions{Port: 10, Confirmed: true})
pending, err := client.ListPendingCommands(ctx, dev.DeviceID)
retracted, err := client.ClearCommands(ctx, dev.DeviceID)

// Moving a device: same bearer, and a device class on the destination network.
dev, err = client.UpdateDevice(ctx, connectivity.UpdateDeviceRequest{
    DeviceID: dev.DeviceID, NetworkID: otherNetworkID, DeviceClassID: otherClassID,
})
```

### What `Client` reaches

| Area | Methods |
|---|---|
| Networks | `CreateNetwork` `ListNetworks` `GetNetwork` `UpdateNetwork` `DeleteNetwork` `GetNetworkOverview` `GetNetworkApplication` |
| Device classes | `CreateDeviceClass` `ListDeviceClasses` `UpdateDeviceClass` `DeleteDeviceClass` |
| Devices | `RegisterDevice` `GetDevice` `GetDeviceState` `UpdateDevice` `DeleteDevice` `ListDevices` `SetDeviceSecret` |
| Cellular gate | `DeviceAccess` `SetDeviceKey` `ClearDeviceKey` `EnableDevice` `DisableDevice` |
| Diagnosis | `DeviceFrames` `DeviceLogs` `GetDeviceActivity` `GetDeviceMetrics` `GetDeviceRadioMetrics` `InspectPayload` |
| Commands | `SendCommand` `ListPendingCommands` `ClearCommands` |
| Access points | `RegisterAccessPoint` `ListAccessPoints` `GetAccessPoint` `UpdateAccessPoint` `DeleteAccessPoint` `GetAccessPointRadioMetrics` |
| Keys | `CreateDeviceKey` `ListDeviceKeys` `RevokeDeviceKey` |
| API keys | `CreateApiKey` `ListApiKeys` `RevokeApiKey` |
| Destinations (integrations) | `CreateIntegration` `ListIntegrations` `UpdateIntegration` `DeleteIntegration` `TestIntegration` `Deliveries` `RetryDelivery` |
| Your platforms | `ListDestinations` `CreateDestination` `UpdateDestination` `DeleteDestination` |
| Watches | `CreateEventRule` `ListEventRules` `DeleteEventRule` `NetworkAlerts` |
| Billing | `ListPlans` `GetSubscription` `ListInvoices` `ListCheckoutMethods` `GetUsage` |
| Business | `GetOverview` `GetAuditLog` `ListApplications` `ListApplicationDevices` |

`staff.go` holds the routes that need a platform-staff principal rather than a
customer one (`ListUnclaimedDevices`, `StaffBillingSummary`, the plan and
assignment calls). A customer token is refused there with 403.

### Three shapes worth knowing before you call

**Credentials are returned exactly once.** `CreateApiKey` is the only call that
ever carries a `Token`; afterwards only its `Prefix` is readable. `SetDeviceKey`,
`SetDeviceSecret` and a destination's `AuthSecret` have no read-back at all.

**Two calls report failure with a `nil` error.** `TestIntegration` and
`RetryDelivery` answer 200 whether or not the delivery landed: the call
succeeded, the delivery is what failed. Branch on `res.Success`, not on `err`.

```go
res, err := client.TestIntegration(ctx, networkID, integrationID)
if err != nil { return err }          // we could not ask
if !res.Success { ... }               // we asked; your endpoint refused
```

**Paging backwards, not by page.** `DeviceFrames` and `DeviceLogs` take a
`FrameQuery{Limit, Before}` and answer with `HasMore`. Do not re-derive the end
of history by counting rows against your limit — a page is cut on a whole
second, so a short page can still have history behind it.

## PlatformClient — one network's API key

```go
p, err := connectivity.NewPlatformFromEnv("CELLULAR")   // CONNECTIVITY.CELLULAR.API_KEY
net, err := p.Network(ctx)                              // fails loudly if the key is for another network

dev, err := p.CreateDevice(ctx, connectivity.CreatePlatformDeviceRequest{
    DeviceID: "867329050123456", Name: "Meter 4",
    DeviceClassID: classID, Key: deviceKeyHex, Enabled: true,
})

dev, err = p.UpdateDevice(ctx, dev.DeviceID, connectivity.UpdatePlatformDeviceRequest{Name: "Meter 5"})

access, err := p.SetKey(ctx, dev.DeviceID, newKeyHex)   // no read-back
access, err = p.Enable(ctx, dev.DeviceID)               // conflict names what is missing
frames, err := p.Frames(ctx, dev.DeviceID, connectivity.FrameQuery{Limit: 50})
```

| Variable | Meaning |
|---|---|
| `CONNECTIVITY.BASE_URL` | API root including `/v1` |
| `CONNECTIVITY.<NETWORK>.API_KEY` | the network's API key |
| `CONNECTIVITY.<NETWORK>.NETWORK_ID` | checked on first use, so a key in the wrong variable fails loudly |
| `CONNECTIVITY.BUSINESS_ID` | checked, not sent |

## Receiving messages

Messages are delivered to you, not polled. `MessageHandler` is the endpoint you register as your
platform URL; it authenticates every request and refuses everything when the secret is empty
rather than accepting anything.

```go
http.Handle("/connectivity/messages", connectivity.MessageHandler(
    "X-Message-Secret", os.Getenv("MESSAGE_SECRET"),
    func(ctx context.Context, up connectivity.Message) (*connectivity.Reply, error) {
        reading := decode(up.Payload)          // your bytes, undecoded on the way to you
        return &connectivity.Reply{Payload: ack(reading)}, nil
    }))
```

On the cellular bearer the `Reply` is written back on the socket the device is holding open. An
error from your handler answers 500, and the reading is then recorded on our side as received but
not accepted by you.

## Errors

Non-2xx answers come back as `*connectivity.APIError` carrying `Status`, `Code` and `Message`.
`connectivity.IsNotFound(err)` and `connectivity.IsConflict(err)` cover the two cases callers
branch on most (a conflict is e.g. a device with no active key, or a class still in use).

## Deliberate absences

- **No `GetDeviceSecret`, no `GetKey`.** The platform has no read-back route: a credential that can
  be fetched leaks through any read-only role. Hold your own copy if you need one.
- **`GetUsage` takes no date range** — the endpoint always reports the current billing period.
- **Messages are delivered, not polled.** There is no "list messages" call here.
- **No `GetDeviceSecret`.** Same reason as the key: there is no read-back route.

## Releases

This repository is published from the Akoben monorepo; `main` tracks the platform's source of
truth. Pin a tag. Breaking changes take a new major version, and a tag is never moved.
