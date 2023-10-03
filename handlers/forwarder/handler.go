package forwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"
)

var errNoLambdaContext = fmt.Errorf("no lambda context found")

type S3Client interface {
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Handler struct {
	DestinationURI *url.URL
	S3Client       S3Client
	Logger         logr.Logger
}

// GetCopyObjectInput constructs the input struct for CopyObject.
func GetCopyObjectInput(source, destination *url.URL) *s3.CopyObjectInput {
	if source == nil || destination == nil {
		return nil
	}

	var (
		bucket     = destination.Host
		copySource = fmt.Sprintf("%s%s", source.Host, source.Path)
		// empty string as base strips the leading slash
		key = strings.TrimLeft(fmt.Sprintf("%s%s", strings.Trim(destination.Path, "/"), source.Path), "/")
	)

	return &s3.CopyObjectInput{
		Bucket:     &bucket,
		CopySource: &copySource,
		Key:        &key,
	}
}

func GetRecordInput(lctx *lambdacontext.LambdaContext, destination *url.URL, r io.Reader) *s3.PutObjectInput {
	if lctx == nil || destination == nil {
		return nil
	}

	key := strings.TrimLeft(fmt.Sprintf("%s/forwarder/%s/%s", strings.Trim(destination.Path, "/"), lctx.InvokedFunctionArn, lctx.AwsRequestID), "/")

	return &s3.PutObjectInput{
		Bucket:      &destination.Host,
		Key:         &key,
		Body:        r,
		ContentType: aws.String("application/x-ndjson"),
	}
}

func (h *Handler) GetDestinationRegion(ctx context.Context, client s3.HeadBucketAPIClient) (string, error) {
	region, err := manager.GetBucketRegion(ctx, client, h.DestinationURI.Host)
	if err != nil {
		return "", fmt.Errorf("failed to get region: %w", err)
	}
	return region, nil
}

func (h *Handler) Handle(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	lctx, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return response, errNoLambdaContext
	}

	logger := h.Logger.WithValues("requestId", lctx.AwsRequestID)

	logger.V(3).Info("handling request")
	defer func() {
		if err != nil {
			logger.Error(err, "failed to process request", "payload", request)
		}
	}()

	var records bytes.Buffer
	defer func() {
		if err == nil {
			logger.V(3).Info("recording results")
			_, err = h.S3Client.PutObject(ctx, GetRecordInput(lctx, h.DestinationURI, &records))
		}
	}()

	encoder := json.NewEncoder(&records)

	for _, record := range request.Records {
		m := &SQSMessage{SQSMessage: record}
		for _, sourceURI := range m.GetObjectCreated() {
			copyInput := GetCopyObjectInput(sourceURI, h.DestinationURI)
			if _, cerr := h.S3Client.CopyObject(ctx, copyInput); cerr != nil {
				logger.Error(cerr, "error copying file")
				m.ErrorMessage = cerr.Error()
				response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
					ItemIdentifier: record.MessageId,
				})
				break
			}
		}
		if err := encoder.Encode(m); err != nil {
			return response, fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return response, nil
}

func New(cfg *Config) (*Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	u, _ := url.ParseRequestURI(cfg.DestinationURI)

	h := &Handler{
		DestinationURI: u,
		S3Client:       cfg.S3Client,
		Logger:         logr.Discard(),
	}

	if cfg.Logger != nil {
		h.Logger = *cfg.Logger
	}

	return h, nil
}
