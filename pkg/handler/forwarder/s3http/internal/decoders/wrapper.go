package decoders

import (
	"fmt"
	"io"

	gzip "github.com/klauspost/pgzip"
)

var wrappers = map[string]Wrapper{
	"": func(fn DecoderFactory) DecoderFactory {
		return fn
	},
	"gzip": GzipWrapper,
}

type Wrapper func(DecoderFactory) DecoderFactory

func GzipWrapper(fn DecoderFactory) DecoderFactory {
	return func(r io.Reader, params map[string]string) Decoder {
		gr, err := gzip.NewReader(r)
		if err != nil {
			return &errorDecoder{fmt.Errorf("failed to read gzip: %w", err)}
		}

		pr, pw := io.Pipe()
		go func() {
			var copyErr error
            defer func() {
                if closeErr := gr.Close(); closeErr != nil && copyErr == nil {
                    pw.CloseWithError(fmt.Errorf("failed to close gzip reader: %w", closeErr))
                    return
                }
                pw.CloseWithError(copyErr)
            }()
            _, copyErr = io.Copy(pw, gr)
        }()

		return fn(pr, params)
	}
}
