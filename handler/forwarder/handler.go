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

	"github.com/observeinc/aws-sam-testing/handler"
)

var errNoLambdaContext = fmt.Errorf("no lambda context found")

type S3Client interface {
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Handler struct {
	handler.Mux
	MaxFileSize    int64
	DestinationURI *url.URL
	LogPrefix      string
	S3Client       S3Client
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

func GetLogInput(lctx *lambdacontext.LambdaContext, prefix string, destination *url.URL, r io.Reader) *s3.PutObjectInput {
	if lctx == nil || destination == nil {
		return nil
	}

	key := strings.TrimLeft(fmt.Sprintf("%s/%s%s/%s", strings.Trim(destination.Path, "/"), prefix, lctx.InvokedFunctionArn, lctx.AwsRequestID), "/")

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

	logger := logr.FromContextOrDiscard(ctx)

	var messages bytes.Buffer
	defer func() {
		if err == nil {
			logger.V(3).Info("logging messages")
			_, err = h.S3Client.PutObject(ctx, GetLogInput(lctx, h.LogPrefix, h.DestinationURI, &messages))
		}
	}()

	encoder := json.NewEncoder(&messages)

	for _, record := range request.Records {
		m := &SQSMessage{SQSMessage: record}
		copyRecords := m.GetObjectCreated()
		for _, copyRecord := range copyRecords {
			if copyRecord.Size != nil && h.MaxFileSize > 0 && *copyRecord.Size > h.MaxFileSize {
				logger.V(1).Info("object size exceeds the maximum file size limit; skipping copy",
					"max", h.MaxFileSize, "size", *copyRecord.Size, "uri", copyRecord.URI)
				// Log a warning and skip this object by continuing to the next iteration
				continue
			}

			sourceURL, err := url.Parse(copyRecord.URI)
			if err != nil {
				logger.Error(err, "error parsing source URI", "SourceURI", copyRecord.URI)
				continue
			}

			copyInput := GetCopyObjectInput(sourceURL, h.DestinationURI)
			if _, cerr := h.S3Client.CopyObject(ctx, copyInput); cerr != nil {
				logger.Error(cerr, "error copying file", "SourceURI", copyRecord.URI)
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

func New(cfg *Config) (h *Handler, err error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	u, _ := url.ParseRequestURI(cfg.DestinationURI)

	h = &Handler{
		DestinationURI: u,
		LogPrefix:      cfg.LogPrefix,
		S3Client:       cfg.S3Client,
		MaxFileSize:    cfg.MaxFileSize,
	}

	if cfg.Logger != nil {
		h.Mux.Logger = *cfg.Logger
	}

	if err := h.Mux.Register(h.Handle); err != nil {
		return nil, fmt.Errorf("failed to register functions: %w", err)
	}

	return h, nil
}
