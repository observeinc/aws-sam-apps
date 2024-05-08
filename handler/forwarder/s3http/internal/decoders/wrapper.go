package decoders

import (
	"compress/gzip"
	"fmt"
	"io"
)

var wrappers = map[string]Wrapper{
	"": func(fn DecoderFactory) DecoderFactory {
		return fn
	},
	"gzip": GzipWrapper,
}

type Wrapper func(DecoderFactory) DecoderFactory

func GzipWrapper(fn DecoderFactory) DecoderFactory {
	return func(r io.Reader) Decoder {
		gr, err := gzip.NewReader(r)
		if err != nil {
			return &errorDecoder{fmt.Errorf("failed to read gzip: %w", err)}
		}
		defer gr.Close()
		return fn(gr)
	}
}
