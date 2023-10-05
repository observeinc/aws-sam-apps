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

func (m *SQSMessage) GetObjectCreated() (uris []*url.URL) {
	message := []byte(m.Body)

	var snsEntity events.SNSEntity
	if err := json.Unmarshal(message, &snsEntity); err == nil {
		if snsEntity.Subject == "Amazon S3 Notification" {
			uris = append(uris, processS3Event([]byte(snsEntity.Message))...)
		}
	}

	if len(uris) == 0 {
		uris = append(uris, processS3Event(message)...)
	}

	if len(uris) == 0 {
		uris = append(uris, processCopyEvent(message)...)
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

func processS3Event(message []byte) (uris []*url.URL) {
	var s3records events.S3Event
	err := json.Unmarshal(message, &s3records)

	if err == nil {
		for _, record := range s3records.Records {
			if strings.HasPrefix(record.EventName, "ObjectCreated") {
				if u := getS3URI(record.S3.Bucket.Name, record.S3.Object.Key); u != nil {
					uris = append(uris, u)
				}
			}
		}
	}
	return
}

type CopyEvent struct {
	Copy []CopyRecord `json:"copy"`
}

type CopyRecord struct {
	URI string `json:"uri"`
}

func processCopyEvent(message []byte) (uris []*url.URL) {
	var copyEvent CopyEvent
	err := json.Unmarshal(message, &copyEvent)

	if err == nil {
		for _, record := range copyEvent.Copy {
			if u, err := url.ParseRequestURI(record.URI); err == nil {
				uris = append(uris, u)
			}
		}
	}
	return
}
