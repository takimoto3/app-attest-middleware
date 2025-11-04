# App Attest Middleware

[![Go Report Card](https://goreportcard.com/badge/github.com/takimoto3/app-attest-middleware)](https://goreportcard.com/report/github.com/takimoto3/app-attest-middleware)

A Go library that provides HTTP middleware and handlers to simplify server-side verification for Apple’s **App Attest** feature.  
It manages both the attestation and assertion flows, allowing you to focus on your application's core logic.

It helps your backend ensure that requests come from legitimate instances of your iOS app, 
reducing the risk of spoofed or tampered clients.

## System Requirements

- Go 1.24 or newer

## Installation

```sh
go get github.com/takimoto3/app-attest-middleware@latest
```

## Features

-   **HTTP Handlers for Attestation**: Provides handlers for the App Attest attestation flow (challenge and verification).
-   **HTTP Middleware for Assertion**: Provides middleware for verifying App Attest assertions on incoming requests.
-   **Extensible Plugin System**: Customize the behavior of the middleware by implementing `AdapterPlugin` interfaces for both attestation and assertion.
-   **Structured Logging with Request ID**: Uses the standard `slog` library for structured, context-aware logging. It automatically injects a request ID (from the `x-request-id` header or a newly generated one) into the context for improved traceability across requests.

## Overview

Apple’s App Attest helps verify that incoming requests originate from a legitimate instance of your app, protecting your backend from fraudulent access.

This library handles the server-side portion of the App Attest verification flow.

## Architecture / Components

This package provides two primary components: `Handler` and `Middleware`.

| Component  | Responsibility |
| :---------- | :------------- |
| `Handler`    | Manages the one-time **attestation** process to verify the app’s integrity. |
| `Middleware` | Manages the recurring **assertion** process to validate user requests. |

The overall architecture is as follows:

```
┌─────────────────────────────────────┐          ┌─────────────────────────────────────┐
│          Server (Handler)           │          │         Server (Middleware)         │
│  Handler                            │          │  Middleware                         │
│     │                               │          │     │                               │
│     ▼                               │          │     ▼                               │
│   Adapter <── AttestationService    │          │   Adapter <── AssertionService      │
│     │                               │          │     │                               │
│     ▼                               │          │     ▼                               │
│   Plugin                            │          │   Plugin                            │
└─────────────────────────────────────┘          └─────────────────────────────────────┘
```

-   **Handler**: Configurable with any HTTP router (e.g., `http.ServeMux`) to handle attestation endpoints such as `/attest/challenge` and `/attest/verify`.
-   **Middleware**: Wraps existing handlers to verify assertions before request handling.
-   **Adapter/Plugin**: Abstract the business logic and data persistence, allowing you to customize the behavior.

## Initialization (shared by Handler and Middleware)

**Important**: Before using the handler or middleware, you must initialize the request ID generator. This is a common step for both components. If the `x-request-id` header is missing, a new one will be generated automatically.

### Using `sonyflake`

To use `sonyflake` for ID generation, you must initialize it at the beginning of your application:

```go
import "github.com/takimoto3/app-attest-middleware/requestid"
import "github.com/sony/sonyflake/v2"

func main() {
    requestid.UseSnowFlake(sonyflake.Settings{})
    // ... rest of your main function
}
```
### Using UUID (v4 or v6)

You can use UUIDs as request IDs.  
Version 4 (random) is used by default, or you can use version 6 for time-ordered IDs.

```go
import "github.com/takimoto3/app-attest-middleware/requestid"

func main() {
    requestid.UseUUID()   // Default: UUIDv4 (random)
    // or
    requestid.UseUUIDv6() // UUIDv6 (time-ordered)
}
```

### Using a Custom Request ID Generator

You can also provide your own implementation of the `Generator` interface:

```go
import "github.com/takimoto3/app-attest-middleware/requestid"

type MyCustomGenerator struct{}

func (g *MyCustomGenerator) NextID() (string, error) {
    // Implement your custom ID generation logic here
    return "my-custom-id", nil
}

func main() {
    requestid.UseGenerator(&MyCustomGenerator{})
    // ... rest of your main function
}
```

## Handler Usage

The handler is responsible for the one-time attestation flow that establishes trust between your app and server.

### 1. Create an AppAttestHandler

First, create an `AppAttestHandler` using the `NewAppAttestHandler` function. You need to provide an `Adapter` that implements the business logic. The `Adapter` in turn uses a `Plugin` to interact with your application's data store.

```go
// main.go

// Initialization
requestid.UseSnowFlake(sonyflake.Settings{})

// 1. Create the AttestationService using the underlying app-attest package
pool, _ := certs.LoadCertFiles("certs/Apple_App_Attestation_Root_CA.pem")
attestationService := attest.NewAttestationService(pool, "<TEAM ID>.<BUNDLE ID>")

// 2. Implement the handler.AdapterPlugin interface
type MyAttestationPlugin struct{}
// ... implementation of the plugin methods ...

// 3. Create the handler Adapter
attestationPlugin := &MyAttestationPlugin{}
attestationAdapter := handler.NewAttestationAdapter(logger, attestationService, attestationPlugin)

// 4. Create the AppAttestHandler
attestHandler := handler.NewAppAttestHandler(logger, attestationAdapter)
```

### 2. Register Routes

Register the `Verify` and `NewChallenge` handlers in your router.

```go
// main.go
mux := http.NewServeMux()
// Register the handlers to your desired endpoints
mux.HandleFunc("/attest/verify", attestHandler.Verify) // Example path
mux.HandleFunc("/attest/challenge", attestHandler.NewChallenge) // Example path
```

### 3. Customize Hooks (Optional)

You can extend or override default behaviors using lifecycle hooks on both handlers and middleware.

-   **`handler.VerifyHooks`**: Called during the attestation verification process.
    -   `Setup`: Called before verification.
    -   `Success`: Called on successful verification.
    -   `Failed`: Called on verification failure.
-   **`handler.NewChallengeHooks`**: Called during the challenge generation process.
    -   `Setup`: Called before challenge generation.
    -   `Success`: Called on successful challenge generation.
    -   `Failed`: Called on challenge generation failure.

You can extend or override default behaviors using lifecycle hooks on the handler.

```go
// main.go
attestHandler.VerifyHooks.Success = func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNoContent)
}
attestHandler.NewChallengeHooks.Success = func(w http.ResponseWriter, r *http.Request, challenge string) {
    json.NewEncoder(w).Encode(map[string]string{"challenge": challenge})
}
```
### Endpoints Summary

-   **Attestation Verification**: The `attestHandler.Verify` method can be registered to any desired endpoint.
-   **Challenge Generation**: The `attestHandler.NewChallenge` method can be registered to any desired endpoint.
-   **Protected Endpoints**: Any endpoint wrapped with the `AssertionMiddleware`.

Example route registration:
```go
mux.HandleFunc("/attest/verify", attestHandler.Verify)
mux.HandleFunc("/attest/challenge", attestHandler.NewChallenge)
```
These paths are only examples — you can freely configure them in your router or application settings.


## Middleware Usage

The middleware protects your existing HTTP endpoints by verifying App Attest assertions on every request, ensuring that only verified app instances can access your APIs.

### Middleware Configuration

The `middleware.Config` struct allows you to configure the behavior of the assertion middleware:

```go
type Config struct {
	BodyLimit       int64  // Maximum size of the request body in bytes. Defaults to 10MB if not set.
	AttestationURL  string // URL to redirect to if attestation is required.
	NewChallengeURL string // URL to redirect to if a new challenge is needed.
}
```

-   **`BodyLimit`**: Sets the maximum allowed size for the request body in bytes. Requests with bodies exceeding this limit will be rejected with a `400 Bad Request`. If not explicitly set, it defaults to 10MB.
-   **`AttestationURL`**: The URL where the client should be redirected if the App Attest attestation is required (i.e., the client has not yet attested or their attestation is invalid).
-   **`NewChallengeURL`**: The URL where the client should be redirected if a new assertion challenge is needed. If this is empty, the middleware will attempt to use the `Referer` header, or default to `/`.

### 1. Create an AssertionMiddleware

Similar to the handler, you create an `AssertionMiddleware` with an `Adapter` and a `Plugin`.

```go
// main.go

import "github.com/takimoto3/app-attest-middleware/middleware"

// Initialization
requestid.UseSnowFlake(sonyflake.Settings{})

// 1. Implement the middleware.AdapterPlugin interface
type MyAssertionPlugin struct{}
// ... implementation of the plugin methods ...

// 2. Create the Middleware Adapter
assertionPlugin := &MyAssertionPlugin{}
assertionAdapter := middleware.NewAssertionAdapter(logger, "<TEAM ID>.<BUNDLE ID>", assertionPlugin)

// 3. Create the AssertionMiddleware
assertionMiddleware := middleware.NewAssertionMiddleware(
    logger,
    middleware.Config{
        BodyLimit:       5 << 20, // Example: 5MB limit for request body
        AttestationURL:  "/attest/verify",
        NewChallengeURL: "/attest/challenge",
    },
    assertionAdapter,
)
```

### 2. Wrap an Existing Handler

Wrap your existing handlers using the middleware’s `Use` method. This ensures that each request is verified before reaching your handler.

```go
// main.go
helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, you have been successfully asserted!")
})

mux.Handle("/hello", assertionMiddleware.Use(helloHandler))
```

## See Also

- [Establishing your app’s integrity (Apple Developer Documentation)](https://developer.apple.com/documentation/devicecheck/establishing-your-app-s-integrity)
- [Validating apps that connect to your server (Apple Developer Documentation)](https://developer.apple.com/documentation/devicecheck/validating-apps-that-connect-to-your-server)

## License

App Attest Middleware is licensed under the MIT License. See [LICENSE](./LICENSE) for details.
