// Package xrayotel is the part of the OpenTelemetry Collector AWS X-Ray
// translator that the poller needs to turn a segment document into OTLP.
//
// Pinned to opentelemetry-collector-contrib cd49174281673a06a5d78f3e5ad994a1bbe38964.
// The UDP receiver, X-Ray API client, telemetry sender, and tests are not copied.
// Semantic-convention attribute names are inlined so this copy does not require
// a newer OpenTelemetry module than the rest of the repo.
package xrayotel
