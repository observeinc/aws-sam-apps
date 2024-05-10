package decoders

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

func CSVDecoderFactory(params map[string]string) DecoderFactory {
	return func(r io.Reader) Decoder {
		buffered := bufio.NewReader(r)
		csvDecoder := &CSVDecoder{
			Reader:   csv.NewReader(buffered),
			buffered: buffered,
		}
		csvDecoder.Reader.FieldsPerRecord = -1

		comma := ','
		if params["delimiter"] == "space" {
			comma = ' '
		}
		csvDecoder.Reader.Comma = comma
		return csvDecoder
	}
}

func VPCFlowLogDecoderFactory(_ map[string]string) DecoderFactory {
	return CSVDecoderFactory(map[string]string{
		"delimiter": "space",
	})
}

type CSVDecoder struct {
	*csv.Reader
	buffered *bufio.Reader
	header   []string
	sync.Once
}

func (dec *CSVDecoder) Decode(v any) error {
	var err error
	dec.Once.Do(func() {
		dec.header, err = dec.Read()
		// After the header is determined, we can allow the csv.Reader to reuse
		// a []string for every subsequent record.
		dec.Reader.ReuseRecord = true
	})
	if err != nil {
		return fmt.Errorf("failed to decode header: %w", err)
	}

	record, err := dec.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to decode record: %w", err)
	}

	var buf bytes.Buffer

	buf.WriteString("{")
	for i, colName := range dec.header {
		if i < len(record) && record[i] != "" {
			if buf.Len() != 1 {
				buf.WriteString(", ")
			}
			if _, err := buf.WriteString(`"` + colName + `": "` + record[i] + `"`); err != nil {
				return fmt.Errorf("failed to write to buffer: %w", err)
			}
		}
	}
	buf.WriteString("}")

	if err := json.Unmarshal(buf.Bytes(), v); err != nil {
		return fmt.Errorf("failed to decode CSV: %w", err)
	}
	return nil
}

// More checks if there is more input.
func (dec *CSVDecoder) More() bool {
	_, err := dec.buffered.Peek(1)
	return err != io.EOF
}
