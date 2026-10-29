package decoders

import (
	"encoding/json"
	"fmt"
	"io"
)

type ConfigurationItem struct {
	AccountID                     *string         `json:"awsAccountId,omitempty"`
	NotificationAccountID         *string         `json:"accountId,omitempty"`
	ARN                           *string         `json:"ARN,omitempty"`
	AvailabilityZone              *string         `json:"availabilityZone,omitempty"`
	AwsRegion                     *string         `json:"awsRegion,omitempty"`
	Configuration                 json.RawMessage `json:"configuration,omitempty"`
	ConfigurationItemCaptureTime  *string         `json:"configurationItemCaptureTime,omitempty"`
	ConfigurationItemDeliveryTime *string         `json:"configurationItemDeliveryTime,omitempty"`
	ConfigurationItemMD5Hash      *string         `json:"configurationItemMD5Hash,omitempty"`
	ConfigurationItemStatus       *string         `json:"configurationItemStatus,omitempty"`
	ConfigurationItemVersion      *string         `json:"configurationItemVersion,omitempty"`
	ConfigurationStateID          *int            `json:"configurationStateId,omitempty"`
	ResourceCreationTime          *string         `json:"resourceCreationTime,omitempty"`
	ResourceID                    *string         `json:"resourceId,omitempty"`
	ResourceName                  *string         `json:"resourceName,omitempty"`
	ResourceType                  *string         `json:"resourceType,omitempty"`
	SupplementaryConfiguration    json.RawMessage `json:"supplementaryConfiguration,omitempty"`
	Tags                          json.RawMessage `json:"tags,omitempty"`
}

type ConfigurationDiff struct {
	ConfigurationItem        *ConfigurationItem `json:"configurationItem,omitempty"`
	ConfigurationItemDiff    json.RawMessage    `json:"configurationItemDiff,omitempty"`
	MessageType              *string            `json:"messageType,omitempty"`
	NotificationCreationTime *string            `json:"notificationCreationTime,omitempty"`
	RecordVersion            *string            `json:"recordVersion,omitempty"`
}

// ConfigurationItemDecoderFactory expands an AWS Config snapshot, retains the
// fields accepted by the entity endpoint, and skips resource types for which
// AWS Config does not provide an ARN. The entity schema requires an ARN, so
// including those records would reject the containing HTTP batch.
func ConfigurationItemDecoderFactory(r io.Reader, params map[string]string) Decoder {
	return &configurationItemDecoder{decoder: NestedJSONDecoderFactory(r, params)}
}

type configurationItemDecoder struct {
	decoder Decoder
	next    json.RawMessage
	err     error
}

func (d *configurationItemDecoder) More() bool {
	if d.err != nil || d.next != nil {
		return true
	}
	for d.decoder.More() {
		var item ConfigurationItem
		if err := d.decoder.Decode(&item); err != nil {
			d.err = fmt.Errorf("failed to decode AWS Config item: %w", err)
			return true
		}
		if item.ARN == nil || *item.ARN == "" {
			continue
		}
		data, err := json.Marshal(&item)
		if err != nil {
			d.err = fmt.Errorf("failed to marshal AWS Config item: %w", err)
			return true
		}
		d.next = data
		return true
	}
	return false
}

func (d *configurationItemDecoder) Decode(v any) error {
	if !d.More() {
		return io.EOF
	}
	if d.err != nil {
		return d.err
	}
	data := d.next
	d.next = nil
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal AWS Config item: %w", err)
	}
	return nil
}
