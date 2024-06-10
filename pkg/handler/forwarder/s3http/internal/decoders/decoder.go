package decoders

import (
	"errors"
	"fmt"
	"io"
	"mime"
)

var (
	ErrUnsupportedContentEncoding = errors.New("content encoding not supported")
	ErrUnsupportedContentType     = errors.New("content type not supported")
)

var decoders = map[string]DecoderFactory{
	"":                                       JSONDecoderFactory,
	"application/json":                       JSONDecoderFactory,
	"application/x-csv":                      CSVDecoderFactory,
	"application/x-ndjson":                   JSONDecoderFactory,
	"text/plain":                             TextDecoderFactory,
	"text/csv":                               CSVDecoderFactory,
	"application/x-aws-cloudwatchlogs":       CloudWatchLogsDecoderFactory,
	"application/x-aws-cloudwatchmetrics":    JSONDecoderFactory,
	"application/x-aws-config":               FilteredNestedJSONDecoderFactory(ConfigurationItem{}),
	"application/x-aws-change":               FilteredJSONDecoderFactory(ConfigurationDiff{}),
	"application/x-aws-cloudtrail":           NestedJSONDecoderFactory,
	"application/x-aws-sqs":                  JSONDecoderFactory,
	"application/x-aws-vpcflowlogs":          SSVDecoderFactory,
	"application/x-aws-elasticloadbalancing": SSVDecoderFactory,
}

type Decoder interface {
	More() bool
	Decode(any) error
}

type (
	DecoderFactory func(io.Reader, map[string]string) Decoder
)

func Get(contentEncoding, contentType string, r io.Reader) (Decoder, error) {
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

	return wrapper(decoder)(r, params), nil
}
