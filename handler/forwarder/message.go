package forwarder

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type SQSMessage struct {
	events.SQSMessage
	ErrorMessage string `json:"error,omitempty"`
}

// ObjectRecord includes the object URI and an optional size
type ObjectRecord struct {
	URI  *url.URL
	Size *int64
}

type CopyRecord struct {
	URI  string `json:"uri"`
	Size *int64 `json:"size,omitempty"`
}

type CopyEvent struct {
	Copy []CopyRecord `json:"copy"`
}

func (m *SQSMessage) GetObjectCreated() (objectRecords []ObjectRecord) {
	message := []byte(m.Body)

	var snsEntity events.SNSEntity
	if err := json.Unmarshal(message, &snsEntity); err == nil {
		if snsEntity.Subject == "Amazon S3 Notification" {
			objectRecords = append(objectRecords, processS3Event([]byte(snsEntity.Message))...)
		}
	}

	if len(objectRecords) == 0 {
		objectRecords = append(objectRecords, processS3Event(message)...)
	}

	if len(objectRecords) == 0 {
		objectRecords = append(objectRecords, processCopyEvent(message)...)
	}

	return
}

func getS3URI(bucketName string, objectKey string) *url.URL {
	s := fmt.Sprintf("s3://%s/%s", bucketName, objectKey)
	if u, err := url.ParseRequestURI(s); err == nil {
		return u
	}
	return nil
}

func processS3Event(message []byte) (objectRecords []ObjectRecord) {
	var s3records events.S3Event
	err := json.Unmarshal(message, &s3records)

	if err == nil {
		for _, record := range s3records.Records {
			if strings.HasPrefix(record.EventName, "ObjectCreated") {
				if u := getS3URI(record.S3.Bucket.Name, record.S3.Object.Key); u != nil {
					objectRecords = append(objectRecords, ObjectRecord{URI: u})
				}
			}
		}
	}
	return
}

func processCopyEvent(message []byte) (objectRecords []ObjectRecord) {
	var copyEvent CopyEvent
	err := json.Unmarshal(message, &copyEvent)

	if err == nil {
		for _, record := range copyEvent.Copy {
			if u, err := url.ParseRequestURI(record.URI); err == nil {
				objectRecords = append(objectRecords, ObjectRecord{URI: u, Size: record.Size})
			}
		}
	}
	return
}
