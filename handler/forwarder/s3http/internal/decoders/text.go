package decoders

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func TextDecoderFactory(map[string]string) DecoderFactory {
	return func(r io.Reader) Decoder {
		return &TextDecoder{
			Reader: bufio.NewReader(r),
		}
	}
}

type TextDecoder struct {
	*bufio.Reader
}

func (dec *TextDecoder) Decode(v any) error {
	s, err := dec.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read text: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(`{"text":` + strconv.Quote(s) + `}`)
	if err := json.Unmarshal(buf.Bytes(), v); err != nil {
		return fmt.Errorf("failed to decode text: %w", err)
	}
	return nil
}

// More checks if there is more input.
func (dec *TextDecoder) More() bool {
	_, err := dec.Peek(1)
	return err != io.EOF
}
