// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package translator // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/translator"

import (
	"encoding/json"

	"go.opentelemetry.io/collector/pdata/pcommon"

	awsxray "github.com/observeinc/aws-sam-apps/pkg/xrayotel/awsxray"
)

func addMetadata(meta map[string]map[string]any, attrs pcommon.Map) error {
	for k, v := range meta {
		val, err := json.Marshal(v)
		if err != nil {
			return err
		}
		attrs.PutStr(
			awsxray.AWSXraySegmentMetadataAttributePrefix+k, string(val),
		)
	}
	return nil
}
