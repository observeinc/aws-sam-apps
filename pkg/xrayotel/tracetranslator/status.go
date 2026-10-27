// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tracetranslator // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/tracetranslator"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// StatusCodeFromHTTP takes an HTTP status code and returns the appropriate
// OpenTelemetry status code. Copied from the collector's tracetranslator.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/http.md#status
func StatusCodeFromHTTP(httpStatusCode int64) ptrace.StatusCode {
	if httpStatusCode >= 100 && httpStatusCode < 399 {
		return ptrace.StatusCodeUnset
	}
	return ptrace.StatusCodeError
}
