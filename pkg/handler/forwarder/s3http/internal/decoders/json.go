package decoders

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

func JSONDecoderFactory(r io.Reader, _ map[string]string) Decoder {
	return json.NewDecoder(r)
}

func FilteredJSONDecoderFactory(record any) DecoderFactory {
	return func(r io.Reader, params map[string]string) Decoder {
		return &FilteredDecoder{
			record:  reflect.ValueOf(record),
			decoder: JSONDecoderFactory(r, params),
		}
	}
}

func FilteredNestedJSONDecoderFactory(record any) DecoderFactory {
	return func(r io.Reader, params map[string]string) Decoder {
		return &FilteredDecoder{
			record:  reflect.ValueOf(record),
			decoder: NestedJSONDecoderFactory(r, params),
		}
	}
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

// FilteredDecoder ensures all data we read in conforms to a provided struct
// This is primarily used for complex records that may exceed our maximum observation size.
type FilteredDecoder struct {
	record  reflect.Value
	decoder Decoder
}

func (d *FilteredDecoder) Decode(v any) error {
	// force new struct to zero out fields
	record := reflect.New(d.record.Type()).Interface()
	if err := d.decoder.Decode(record); err != nil {
		return fmt.Errorf("failed to decode record: %w", err)
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return nil
}

func (d *FilteredDecoder) More() bool {
	return d.decoder.More()
}
