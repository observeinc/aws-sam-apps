package main

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/observeinc/aws-sam-testing/model/aws/ec2/marshaller"
	"github.com/stretchr/testify/assert"
)

var (
	PathToEventJsonFile = "../../apps/ec2/events/event.json"
)

func TestHandler(t *testing.T) {
	jsonFile, err := os.Open(PathToEventJsonFile)
	if err != nil {
		t.Fatal("Unable to open event.json")
	}

	fmt.Println("Successfully Opened event.json")

	defer jsonFile.Close()

	jsonStream, _ := io.ReadAll(jsonFile)

	awsEvent, _ := marshaller.UnmarshalEvent(jsonStream)

	t.Run("Successful Request", func(t *testing.T) {
		responseStream, err := handler(nil, awsEvent)
		if err != nil {
			t.Fatal("Everything should be ok")
		}

		responseEvent, _ := marshaller.UnmarshalEvent(responseStream)
		responseEc2Notification := responseEvent.Detail

		assert.NotEmpty(t, responseEvent)
		assert.Equal(t, responseEvent.DetailType, "HelloWorldFunction updated event of EC2 Instance State-change Notification")
		assert.Equal(t, responseEc2Notification.InstanceId, "i-abcd1111")
	})
}
