// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package translator // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/translator"

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	awsxray "github.com/observeinc/aws-sam-apps/pkg/xrayotel/awsxray"
	"github.com/observeinc/aws-sam-apps/pkg/xrayotel/metadata"
	"github.com/observeinc/aws-sam-apps/pkg/xrayotel/tracetranslator"
)

func addHTTP(seg *awsxray.Segment, span ptrace.Span) {
	if seg.HTTP == nil {
		return
	}

	attrs := span.Attributes()
	if req := seg.HTTP.Request; req != nil {
		// https://docs.aws.amazon.com/xray/latest/devguide/xray-api-segmentdocuments.html#api-segmentdocuments-http
		if metadata.ReceiverAwsxrayEmitV1HTTPConventionsFeatureGate.IsEnabled() {
			addString(req.Method, "http.request.method", attrs)
		}
		if !metadata.ReceiverAwsxrayDontEmitV0HTTPConventionsFeatureGate.IsEnabled() {
			addString(req.Method, "http.method", attrs)
		}

		if req.ClientIP != nil {
			// since the ClientIP is not nil, this means that this segment is generated
			// by a server serving an incoming request
			if !metadata.ReceiverAwsxrayDontEmitV0HTTPConventionsFeatureGate.IsEnabled() {
				attrs.PutStr("http.client_ip", *req.ClientIP)
			}
			if metadata.ReceiverAwsxrayEmitV1HTTPConventionsFeatureGate.IsEnabled() {
				attrs.PutStr("client.address", *req.ClientIP)
			}
		}

		addString(req.UserAgent, "http.user_agent", attrs)
		if metadata.ReceiverAwsxrayEmitV1HTTPConventionsFeatureGate.IsEnabled() {
			addString(req.URL, "url.full", attrs)
		}
		if !metadata.ReceiverAwsxrayDontEmitV0HTTPConventionsFeatureGate.IsEnabled() {
			addString(req.URL, "http.url", attrs)
		}
		addBool(req.XForwardedFor, awsxray.AWSXRayXForwardedForAttribute, attrs)
	}

	if resp := seg.HTTP.Response; resp != nil {
		if resp.Status != nil {
			otStatus := tracetranslator.StatusCodeFromHTTP(*resp.Status)
			// in X-Ray exporter, the segment status is set:
			// first via the span attribute, string(conventions.HTTPStatusCodeKey)
			// then the span status. Since we are also setting the span attribute
			// below, the span status code here will not be actually used
			span.Status().SetCode(otStatus)
			if metadata.ReceiverAwsxrayEmitV1HTTPConventionsFeatureGate.IsEnabled() {
				attrs.PutInt("http.response.status_code", *resp.Status)
			}
			if !metadata.ReceiverAwsxrayDontEmitV0HTTPConventionsFeatureGate.IsEnabled() {
				attrs.PutInt("http.status_code", *resp.Status)
			}
		}

		switch val := resp.ContentLength.(type) {
		case string:
			addString(&val, "http.response_content_length", attrs)
		case float64:
			lengthPointer := int64(val)
			addInt64(&lengthPointer, "http.response_content_length", attrs)
		}
	}
}
