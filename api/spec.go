// Package api embeds the OpenAPI contract, the source of truth for the HTTP API.
//
// The server code in internal/oapi is generated from this file, requests are
// validated against it at runtime, and GET /openapi.yaml serves it as is.
package api

import _ "embed"

// Spec is the OpenAPI 3.1 document in api/openapi.yaml.
//
//go:embed openapi.yaml
var Spec []byte
