// godoc.go
// Package appattest provides core functionality for Apple App Attestation verification.
//
// It includes HTTP handlers, middlewares, and request utilities to
// integrate attestation verification into Go web services.
//
// Subpackages:
//   - handler: contains HTTP route handlers for verification endpoints
//   - middleware: provides common middleware like request ID injection
//   - requestid: handles request ID generation and propagation
package appattest
