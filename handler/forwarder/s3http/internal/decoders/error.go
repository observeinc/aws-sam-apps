package decoders

// Helper to return an error on first decode.
type errorDecoder struct {
	Error error
}

func (d *errorDecoder) More() bool {
	return true
}

func (d *errorDecoder) Decode(any) error {
	return d.Error
}
