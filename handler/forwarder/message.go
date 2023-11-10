package forwarder

import (
	"encoding/json"
	"fmt"
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
