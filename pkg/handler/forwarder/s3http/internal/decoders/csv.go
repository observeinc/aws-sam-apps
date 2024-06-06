package decoders

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
)

var ErrUnsupportedDelimiter = errors.New("unsupported delimiter")

func CSVDecoderFactory(r io.Reader, params map[string]string) Decoder {
	buffered := bufio.NewReader(r)
	csvDecoder := &CSVDecoder{
		Reader:   csv.NewReader(buffered),
		buffered: buffered,
	}
	csvDecoder.Reader.FieldsPerRecord = -1

	var delimiter rune
	switch params["delimiter"] {
	case "space":
		delimiter = ' '
	case "tab":
		delimiter = '\t'
	case "comma", "":
		delimiter = ','
	default:
		err := fmt.Errorf("%w: %q", ErrUnsupportedDelimiter, params["delimiter"])
		return &errorDecoder{err}
	}
	csvDecoder.Reader.Comma = delimiter
	return csvDecoder
}

// SSVDecoderFactory handles space separated values.
func SSVDecoderFactory(r io.Reader, params map[string]string) Decoder {
	if _, ok := params["delimiter"]; !ok {
		params["delimiter"] = "space"
	}

	return CSVDecoderFactory(r, params)
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
		for i, h := range dec.header {
			dec.header[i] = strconv.Quote(h)
		}
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

	buf.WriteString(`{`)
	for i, colName := range dec.header {
		if i < len(record) && record[i] != "" {
			if buf.Len() != 1 {
				buf.WriteString(`,`)
			}
			buf.WriteString(colName + `:` + strconv.Quote(record[i]))
		}
	}
	buf.WriteString(`}`)

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
