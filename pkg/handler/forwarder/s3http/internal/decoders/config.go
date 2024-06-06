package decoders

import (
	"encoding/json"
)

type ConfigurationItem struct {
	AccountID                     *string         `json:"awsAccountId,omitempty"`
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
