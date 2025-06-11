package decoders

import (
	"fmt"
	"io"
	"log"

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
			_, copyErr := io.Copy(pw, gr)
			if closeErr := gr.Close(); closeErr != nil {
				log.Printf("failed to close gzip reader: %v", closeErr)
			}
			pw.CloseWithError(copyErr)
		}()

		return fn(pr, params)
	}
}
