package decoders

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSONDecoderFactory(r io.Reader, _ map[string]string) Decoder {
	return json.NewDecoder(r)
}

func NestedJSONDecoderFactory(r io.Reader, _ map[string]string) Decoder {
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
