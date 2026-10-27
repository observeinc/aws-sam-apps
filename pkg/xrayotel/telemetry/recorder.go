// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/telemetry"

// Recorder is the subset of the collector X-Ray telemetry recorder that the
// segment translator calls. The poller does not forward daemon telemetry.
type Recorder interface {
	RecordSegmentsReceived(count int)
	RecordSegmentsRejected(count int)
}

// Nop discards segment counts.
type Nop struct{}

func (Nop) RecordSegmentsReceived(int) {}
func (Nop) RecordSegmentsRejected(int) {}
