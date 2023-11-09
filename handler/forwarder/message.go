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

type CopyRecord struct {
	URI  string `json:"uri"`
	Size *int64 `json:"size,omitempty"`
}

type CopyEvent struct {
	Copy []CopyRecord `json:"copy"`
}

func (m *SQSMessage) GetObjectCreated() (copyRecords []CopyRecord) {
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

func getS3URI(bucketName string, objectKey string) *url.URL {
	s := fmt.Sprintf("s3://%s/%s", bucketName, objectKey)
	if u, err := url.ParseRequestURI(s); err == nil {
		return u
	}
	return nil
}

func processS3Event(message []byte) (copyRecords []CopyRecord) {
	var s3records events.S3Event
	err := json.Unmarshal(message, &s3records)

	if err == nil {
		for _, record := range s3records.Records {
			if strings.HasPrefix(record.EventName, "ObjectCreated") {
				if u := getS3URI(record.S3.Bucket.Name, record.S3.Object.Key); u != nil {
					copyRecords = append(copyRecords, CopyRecord{URI: u.String()})
				}
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
			if record.Size != nil {
				sizeValue := *record.Size // Dereference the pointer to get the value
				copyRecords = append(copyRecords, CopyRecord{URI: record.URI, Size: &sizeValue})
			} else {
				// If size is nil, append the record without a size
				copyRecords = append(copyRecords, CopyRecord{URI: record.URI})
			}
		}
	}

	return
}
