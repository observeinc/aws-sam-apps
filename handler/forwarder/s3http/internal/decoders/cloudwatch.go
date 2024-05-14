package decoders

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-lambda-go/events"
)

func CloudWatchLogsDecoderFactory(r io.Reader, _ map[string]string) Decoder {
	buffered := bufio.NewReader(r)
	return &CloudWatchLogsDecoder{
		buffered: buffered,
		decoder:  json.NewDecoder(buffered),
	}
}

type CloudWatchLogsDecoder struct {
	buffered *bufio.Reader
	decoder  *json.Decoder

	messages [][]byte
}

type CloudWatchLogMessage struct {
	*events.CloudwatchLogsLogEvent
	Owner               string   `json:"owner"`
	LogGroup            string   `json:"logGroup"`
	LogStream           string   `json:"logStream"`
	SubscriptionFilters []string `json:"subscriptionFilters"`
	MessageType         string   `json:"messageType"`
}

// Decode one cloudwatch log message at a time.
// This requires flattening the original event.
func (dec *CloudWatchLogsDecoder) Decode(v any) error {
	for len(dec.messages) == 0 {
		var data events.CloudwatchLogsData
		if err := dec.decoder.Decode(&data); err != nil {
			return fmt.Errorf("failed to decode cloudwatch logs: %w", err)
		}

		for _, logEvent := range data.LogEvents {
			message, err := json.Marshal(&CloudWatchLogMessage{
				CloudwatchLogsLogEvent: &logEvent,
				Owner:                  data.Owner,
				LogGroup:               data.LogGroup,
				LogStream:              data.LogStream,
				SubscriptionFilters:    data.SubscriptionFilters,
				MessageType:            data.MessageType,
			})
			if err != nil {
				return fmt.Errorf("failed to marshal cloudwatch log: %w", err)
			}
			dec.messages = append(dec.messages, message)
		}
	}

	if err := json.Unmarshal(dec.messages[0], v); err != nil {
		return fmt.Errorf("failed to unmarshal cloudwatch log: %w", err)
	}
	dec.messages = dec.messages[1:]
	return nil
}

// More checks if there is more input.
func (dec *CloudWatchLogsDecoder) More() bool {
	if len(dec.messages) > 0 {
		return true
	}
	return dec.decoder.More()
}
