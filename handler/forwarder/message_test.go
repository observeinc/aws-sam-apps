package forwarder_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/observeinc/aws-sam-testing/handler/forwarder"

	"github.com/google/go-cmp/cmp"
)

// helper function for creating int64 pointers from integers (if needed).
func pointerToInt64(val int64) *int64 {
	return &val
}

func TestObjectCreated(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Message  string
		Expected []forwarder.CopyRecord
	}{
		{
			Message: `{
				"messageId": "f0e2c213-b4e4-4057-ab44-6e277503557b",
				"receiptHandle": "AQEBW5p1btfAULoLcmMqNwSkTPNzZbmXIAhtKzbfExKV7ifwdu7Hxq1T6aw+dd10N7fu1C2APfrdEYn2LIVYxn7nNbnGv+XG4ZiRvwf7PfLkNZmIpilQU6Oyq9iqM6n8OGx8Aqk9mvBlOSUyD5PAS2PXyrKNEqXclfI/4VWnapjaOgYyBTsADs9uDxsKL9XoeqUJIUNxvfLXsqknwx3ygC2nu6RjjZtHrVVgxdo5y3pybelpfg21hG8ZphhHtfw86ZmQq/Go3QtGGuJZXFz3mV0GE/CDJEKoFX4WkVWZ4XytI2Wn/3UCASCR4vuSFHMnrXeMhhRmxfpmCRjTY9Tve3hiwffGduxqLfZpj/mwGi216JXhIXZkG0cvxnrSAp2nwCOqms4l1Wyb13gizB7G6S/vcAc/5wYNoK4F9oVC96wgqlM=",
				"body": "{\"Records\":[{\"eventVersion\":\"2.1\",\"eventSource\":\"aws:s3\",\"awsRegion\":\"us-west-2\",\"eventTime\":\"2023-09-19T02:49:21.921Z\",\"eventName\":\"ObjectCreated:Put\",\"userIdentity\":{\"principalId\":\"AWS:AIDA2YN7JF3XJWA6ZRUAE\"},\"requestParameters\":{\"sourceIPAddress\":\"192.184.182.96\"},\"responseElements\":{\"x-amz-request-id\":\"TGAER6XP8FQ1MG9N\",\"x-amz-id-2\":\"akbPx/SRWbpa2bSzQ/0Qv1gV2QuHNI1Y5zL3nDQEj99fAj9YKJ+OEr/x0wLzQ72443De9ufSiRNJQMvb+YAYFkf6Gw9pmwF1\"},\"s3\":{\"s3SchemaVersion\":\"1.0\",\"configurationId\":\"tf-s3-queue-20230829163550234000000002\",\"bucket\":{\"name\":\"my-bucket\",\"ownerIdentity\":{\"principalId\":\"A21JCN1A8EHLG1\"},\"arn\":\"arn:aws:s3:::my-bucket\"},\"object\":{\"key\":\"test.json\",\"size\":16,\"eTag\":\"ed818579e8cee1d812a77f19efa5e56a\",\"versionId\":\"B4uqIbhdKYPsdJ.MkIjfpH5cOzj7332h\",\"sequencer\":\"0065090C31D64277DA\"}}}]}",
				"attributes": {
					"ApproximateReceiveCount": "1",
					"SentTimestamp": "1695091763045",
					"SenderId": "AIDAJFWZWTE5KRAMGW5A2",
					"ApproximateFirstReceiveTimestamp": "1695091763054"
				},
				"messageAttributes": {},
				"md5OfBody": "729cb6421101a5813747e8336f673dab",
				"eventSource": "aws:sqs",
				"eventSourceARN": "arn:aws:sqs:us-west-2:123456789012:my-queue",
				"awsRegion": "us-west-2"
			}`,
			Expected: []forwarder.CopyRecord{
				{
					URI:  "s3://my-bucket/test.json",
					Size: pointerToInt64(16),
				},
			},
		},
		{
			Message: `
			{
				"messageId": "e990e046-8e53-4fdd-8011-379517940223",
				"receiptHandle": "AQEB6vyZ3VqjMpeOzcd1Simmu+Emi7MnOivmOdS1XEayMhO9RVWI3Ft/hn5YLMEjB5VY2nsSDVCh37gBznhNRvx4AxrdkOHrfn7OOvrTJq/3gG6ecjNEDI/5WpIk6zd3a/rXiN8H7crev336hqxtu4hJSVz66XUkRKda1pfmIlzfBaLMzoBB4hxMKrwpQ1y+IaSx/FXUDMYPBo5r8lG3+sIa/7TpFfFpI8mo0tdnAkF2zeyJ8Hk1YDQZn4Y40cOMxltHGRcIsK2HmT5sa2E0AkAGV5Kd97Pb+Bb+j91+6VJaYQuV0SvkutGlq3aI/8SCZbK2CArqh0gJe32eQKHopRXxi/ihikSa7u47+FzlyPGKnooYOvCAc6zO8rDck+IH9wwCvcCBWc71U6KIG9uiRQu0eemaUDV5dpyRbG4Be9Ep7Io=",
				"body": "{\n  \"Type\" : \"Notification\",\n  \"MessageId\" : \"4999b1ce-cf92-56b2-afe2-ec4076695a9f\",\n  \"TopicArn\" : \"arn:aws:sns:us-east-1:123456789012:config-updates\",\n  \"Subject\" : \"Amazon S3 Notification\",\n  \"Message\" : \"{\\\"Records\\\":[{\\\"eventVersion\\\":\\\"2.1\\\",\\\"eventSource\\\":\\\"aws:s3\\\",\\\"awsRegion\\\":\\\"us-east-1\\\",\\\"eventTime\\\":\\\"2023-09-27T23:16:10.232Z\\\",\\\"eventName\\\":\\\"ObjectCreated:Put\\\",\\\"userIdentity\\\":{\\\"principalId\\\":\\\"AWS:AIDA2YN7JF3XJWA6ZRUAE\\\"},\\\"requestParameters\\\":{\\\"sourceIPAddress\\\":\\\"192.184.182.96\\\"},\\\"responseElements\\\":{\\\"x-amz-request-id\\\":\\\"V3DKPBZWVMCKX1C7\\\",\\\"x-amz-id-2\\\":\\\"b6p+8nV3VGEnXTqnl+3rHvb+UzlTBNPS7Woeq6ftOOMXYStTDociNu4MqUk6BW4hkCInFqMEiuBmTtZC5banA2H9ERIGrbd5\\\"},\\\"s3\\\":{\\\"s3SchemaVersion\\\":\\\"1.0\\\",\\\"configurationId\\\":\\\"tf-s3-topic-20230927215049003100000002\\\",\\\"bucket\\\":{\\\"name\\\":\\\"my-bucket\\\",\\\"ownerIdentity\\\":{\\\"principalId\\\":\\\"A21JCN1A8EHLG1\\\"},\\\"arn\\\":\\\"arn:aws:s3:::my-bucket\\\"},\\\"object\\\":{\\\"key\\\":\\\"test.json\\\",\\\"size\\\":25,\\\"eTag\\\":\\\"d0b8560f261410878a68bbe070d81853\\\",\\\"sequencer\\\":\\\"006514B7BA1ECB9CE0\\\"}}}]}\",\n  \"Timestamp\" : \"2023-09-27T23:16:11.551Z\",\n  \"SignatureVersion\" : \"1\",\n  \"Signature\" : \"R6mj5u0KHmZvcHf83dJyTIExMiApmtsBzEw/dpmIjci7rfiqyE2LtjNZMWbSrXYJ366ZPSI4sX87lFdO1dNoAmUoGshRr1/0vCkva7uv1Zi3SkhEBTFOrCrPKQ13LCvG1sOPWeeFtevjAivLNnwSp3B3kjsGMIbrKFPRsqlnUtEBauKE/hwg1jANTKwZvChNVrfxzqcKvz2TpPACQ4QX9ma3lOWMuI/yY63fXgRAD8y6HSkgYB7OPxut7SEOG0maun+KExL2AocRXGBwb2tFkSHI2p1nlg5agvikKCcvspv3xz5PrEt1Bb9ymCS20Od6WETLjMZS2LlVeQhvfYEI5g==\",\n  \"SigningCertURL\" : \"https://sns.us-east-1.amazonaws.com/SimpleNotificationService-01d088a6f77103d0fe307c0069e40ed6.pem\",\n  \"UnsubscribeURL\" : \"https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:123456789012:config-updates:e96ef48f-aba9-4b15-96ba-dd72e4f84b25\"\n}",
				"md5OfBody": "432f4a36bb2e110e6eed0b3557418012",
				"md5OfMessageAttributes": "",
				"attributes": {
					"ApproximateReceiveCount": "1",
					"SentTimestamp": "1695856571581",
					"SenderId": "AIDAIT2UOQQY3AUEKVGXU",
					"ApproximateFirstReceiveTimestamp": "1695856571592"
				},
				"messageAttributes": {},
				"eventSourceARN": "arn:aws:sqs:us-east-1:123456789012:my-queue",
				"eventSource": "aws:sqs",
				"awsRegion": "us-east-1"
			}`,
			Expected: []forwarder.CopyRecord{
				{
					URI:  "s3://my-bucket/test.json",
					Size: pointerToInt64(25),
				},
			},
		},
		{
			Message: `
			{
			  "attributes": {
				"ApproximateFirstReceiveTimestamp": "1696456364266",
				"ApproximateReceiveCount": "1",
				"SenderId": "AIDAJXNJGGKNS7OSV23OI",
				"SentTimestamp": "1696456364253"
			  },
			  "awsRegion": "us-east-1",
			  "body": "{\"copy\": [{\"uri\": \"s3://my-bucket/test.json\"}]}",
			  "eventSource": "aws:sqs",
			  "eventSourceARN": "arn:aws:sqs:us-east-1:123456789012:joao-filedrop-us-east-1",
			  "md5OfBody": "a231c62bd2ab84f63c85549c8eead615",
			  "md5OfMessageAttributes": "",
			  "messageAttributes": {},
			  "messageId": "420659fa-599c-4a2d-97fa-da7ade83edc7",
			  "receiptHandle": "AQEB2kq3hOZLP6rlatTMVY4VOL37Zj7IFEhQeeIJAkZhM5vqCcBwZYgPzTc3QOtLTEg0DIL7okUsbFxz5ba3soihn5wqPM7x8fXuzJ0sBOE1XyYUBSzL5Ot6xjY7SnijCsnMEUc8wYTvx1LfkGkwXqKS4maXA8+R530YEUr1RLZ8EqHYtCG4tI6RU1jd0a0Mzv0DUFOg/NU7TdcMYlL7LjPClFfUoy9Hw/9R9L2aLfpUODQVD6+r86wlKrzzMLUDHw7BYuBXaXGXD/w9KGrCoL1q9IIkzXh0gbiAseC968vIh2xSfFv0l9tokahqPBpL/w6V8awnU9tNUQLafG3WjFzFjB00SuFedbxAhARUjNDGmaFIqoLdUrlYEfkPpVxrfqmwbunCQ0URzOtMMJu2uIp0XA=="
			}`,
			Expected: []forwarder.CopyRecord{
				{
					URI: "s3://my-bucket/test.json",
				},
			},
		},
		{
			Message: `
			{
			  "attributes": {
				"ApproximateFirstReceiveTimestamp": "1696456364266",
				"ApproximateReceiveCount": "1",
				"SenderId": "AIDAJXNJGGKNS7OSV23OI",
				"SentTimestamp": "1696456364253"
			  },
			  "awsRegion": "us-east-1",
			  "body": "{\"copy\": [{\"uri\": \"s3://my-bucket/test.json\",\"size\":12345}]}",
			  "eventSource": "aws:sqs",
			  "eventSourceARN": "arn:aws:sqs:us-east-1:123456789012:joao-filedrop-us-east-1",
			  "md5OfBody": "a231c62bd2ab84f63c85549c8eead615",
			  "md5OfMessageAttributes": "",
			  "messageAttributes": {},
			  "messageId": "420659fa-599c-4a2d-97fa-da7ade83edc7",
			  "receiptHandle": "AQEB2kq3hOZLP6rlatTMVY4VOL37Zj7IFEhQeeIJAkZhM5vqCcBwZYgPzTc3QOtLTEg0DIL7okUsbFxz5ba3soihn5wqPM7x8fXuzJ0sBOE1XyYUBSzL5Ot6xjY7SnijCsnMEUc8wYTvx1LfkGkwXqKS4maXA8+R530YEUr1RLZ8EqHYtCG4tI6RU1jd0a0Mzv0DUFOg/NU7TdcMYlL7LjPClFfUoy9Hw/9R9L2aLfpUODQVD6+r86wlKrzzMLUDHw7BYuBXaXGXD/w9KGrCoL1q9IIkzXh0gbiAseC968vIh2xSfFv0l9tokahqPBpL/w6V8awnU9tNUQLafG3WjFzFjB00SuFedbxAhARUjNDGmaFIqoLdUrlYEfkPpVxrfqmwbunCQ0URzOtMMJu2uIp0XA=="
			}`,
			Expected: []forwarder.CopyRecord{
				{
					URI:  "s3://my-bucket/test.json",
					Size: func() *int64 { var s int64 = 12345; return &s }(),
				},
			},
		},
	}

	for i, tc := range testcases {
		tc := tc
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			var message forwarder.SQSMessage
			if err := json.Unmarshal([]byte(tc.Message), &message); err != nil {
				t.Fatal(err)
			}

			// Directly compare the CopyRecords obtained from GetObjectCreated
			copyRecords := message.GetObjectCreated()
			if diff := cmp.Diff(copyRecords, tc.Expected); diff != "" {
				t.Errorf("unexpected result: %s", diff)
			}
		})
	}
}
