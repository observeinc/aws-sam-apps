package decoders

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSONDecoderFactory(map[string]string) DecoderFactory {
	return func(r io.Reader) Decoder {
		return json.NewDecoder(r)
	}
}

func NestedJSONDecoderFactory(map[string]string) DecoderFactory {
	return func(r io.Reader) Decoder {
		dec := json.NewDecoder(r)
		tok, err := dec.Token()
		for err == nil {
			if v, ok := tok.(json.Delim); ok {
				if v == '[' {
					break
				}
			}
			tok, err = dec.Token()
		}
		if err != nil {
			return &errorDecoder{fmt.Errorf("unexpected token: %w", err)}
		}
		return dec
	}
}
