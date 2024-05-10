package decoders

import (
	"errors"
	"fmt"
	"io"
	"mime"
)

var (
	ErrUnsupportedContentEncoding = errors.New("content encoding not supported: %w")
	ErrUnsupportedContentType     = errors.New("content type not supported: %w")
)

var decoders = map[string]ParameterizedDecoderFactory{
	"":                     JSONDecoderFactory,
	"application/json":     JSONDecoderFactory,
	"application/x-csv":    CSVDecoderFactory,
	"application/x-ndjson": JSONDecoderFactory,
	"text/plain":           TextDecoderFactory,
	"text/csv":             CSVDecoderFactory,
	// "application/x-aws-cloudwatchlogs":    NestedJSONDecoderFactory,
	"application/x-aws-cloudwatchmetrics": JSONDecoderFactory,
	"application/x-aws-config":            NestedJSONDecoderFactory,
	"application/x-aws-change":            JSONDecoderFactory,
	"application/x-aws-cloudtrail":        NestedJSONDecoderFactory,
	"application/x-aws-sqs":               JSONDecoderFactory,
	"application/x-aws-vpcflowlogs":       VPCFlowLogDecoderFactory,
}

type Decoder interface {
	More() bool
	Decode(any) error
}

type (
	ParameterizedDecoderFactory func(params map[string]string) DecoderFactory
	DecoderFactory              func(io.Reader) Decoder
)

func Get(contentEncoding, contentType string) (DecoderFactory, error) {
	wrapper, ok := wrappers[contentEncoding]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedContentEncoding, contentEncoding)
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content type: %w", err)
	}

	decoder, ok := decoders[mediaType]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedContentType, contentType)
	}

	return wrapper(decoder(params)), nil
}
