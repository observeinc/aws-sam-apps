package forwarder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

const (
	configurationItemChangeNotification    = "ConfigurationItemChangeNotification"
	configurationSnapshotDeliveryCompleted = "ConfigurationSnapshotDeliveryCompleted"
)

type SQSMessage struct {
	events.SQSMessage
	ErrorMessage string `json:"error,omitempty"`
}

type CopyRecord struct {
	URI  string `json:"uri"`
	Size *int64 `json:"size,omitempty"`
}

type CopyEvent struct {
	Copy []CopyRecord `json:"copy"`
}

type configNotification struct {
	MessageType string `json:"messageType"`
	S3Bucket    string `json:"s3Bucket"`
	S3ObjectKey string `json:"s3ObjectKey"`
}

type eventBridgeConfigNotification struct {
	Source string          `json:"source"`
	Detail json.RawMessage `json:"detail"`
}

// GetConfigEntityInput extracts either a resource change or an S3 object from
// the AWS Config notifications delivered through SNS, EventBridge, or SQS.
func GetConfigEntityInput(m *events.SQSMessage) (change json.RawMessage, copies []CopyRecord, handled bool) {
	message := []byte(m.Body)
	var snsEntity events.SNSEntity
	if err := json.Unmarshal(message, &snsEntity); err == nil && snsEntity.Message != "" {
		message = []byte(snsEntity.Message)
		handled = true
	}

	var event eventBridgeConfigNotification
	if err := json.Unmarshal(message, &event); err == nil && event.Source == "aws.config" && len(event.Detail) > 0 {
		message = event.Detail
		handled = true
	}

	var notification configNotification
	if err := json.Unmarshal(message, &notification); err != nil {
		return nil, nil, handled
	}
	switch notification.MessageType {
	case configurationItemChangeNotification:
		return append(json.RawMessage(nil), message...), nil, true
	case configurationSnapshotDeliveryCompleted:
		if notification.S3Bucket != "" && notification.S3ObjectKey != "" {
			return nil, []CopyRecord{{URI: fmt.Sprintf("s3://%s/%s", notification.S3Bucket, notification.S3ObjectKey)}}, true
		}
	}
	return nil, nil, handled || notification.MessageType != ""
}

func GetObjectCreated(m *events.SQSMessage) (copyRecords []CopyRecord) {
	message := []byte(m.Body)

	var snsEntity events.SNSEntity
	if err := json.Unmarshal(message, &snsEntity); err == nil {
		if snsEntity.Subject == "Amazon S3 Notification" {
			copyRecords = append(copyRecords, processS3Event([]byte(snsEntity.Message))...)
		}
	}

	if len(copyRecords) == 0 {
		copyRecords = append(copyRecords, processS3Event(message)...)
	}

	if len(copyRecords) == 0 {
		copyRecords = append(copyRecords, processCopyEvent(message)...)
	}

	return
}

func processS3Event(message []byte) (copyRecords []CopyRecord) {
	var s3records events.S3Event
	err := json.Unmarshal(message, &s3records)

	if err == nil {
		for _, record := range s3records.Records {
			if strings.HasPrefix(record.EventName, "ObjectCreated") {
				uri := fmt.Sprintf("s3://%s/%s", record.S3.Bucket.Name, record.S3.Object.Key)

				// Initialize a CopyRecord with URI
				copyRecord := CopyRecord{URI: uri}

				// Only set Size if it's present in the S3 event
				if record.S3.Object.Size != 0 {
					size := record.S3.Object.Size
					copyRecord.Size = &size
				}

				// Append the copyRecord to the slice
				copyRecords = append(copyRecords, copyRecord)
			}
		}
	}
	return
}

func processCopyEvent(message []byte) (copyRecords []CopyRecord) {
	var copyEvent CopyEvent
	err := json.Unmarshal(message, &copyEvent)

	if err == nil {
		for _, record := range copyEvent.Copy {
			copyRecords = append(copyRecords, CopyRecord{URI: record.URI, Size: record.Size})
		}
	}

	return
}
