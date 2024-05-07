package decoders

import (
	"encoding/json"
	"fmt"
	"io"
)

var JSONDecoderFactory = func(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

var NestedJSONDecoderFactory = func(r io.Reader) Decoder {
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
