package forwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
var ErrFileSizeLimitExceeded = errors.New("file size exceeds limit")

type S3Client interface {
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type Handler struct {
	DestinationURI *url.URL
	LogPrefix      string
	S3Client       S3Client
	Logger         logr.Logger
	SizeLimit      int64
}

func (h *Handler) IsObjectSizeWithinLimit(ctx context.Context, source *url.URL) (bool, error) {
	if source == nil {
		return false, fmt.Errorf("source URL is nil")
	}

	// Remove any leading slashes from the source.Path
	key := strings.TrimPrefix(source.Path, "/")

	input := &s3.HeadObjectInput{
		Bucket: aws.String(source.Host),
		Key:    aws.String(key),
	}

	output, err := h.S3Client.HeadObject(ctx, input)

	if err != nil {
		return false, fmt.Errorf("failed to get object head: %w", err)
	}

	if output == nil {
		return false, fmt.Errorf("output is nil")
	}

	return output.ContentLength <= h.SizeLimit, nil
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

	logger := h.Logger.WithValues("requestId", lctx.AwsRequestID)

	logger.V(3).Info("handling request")
	defer func() {
		if err != nil {
			logger.Error(err, "failed to process request", "payload", request)
		}
	}()

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
		for _, sourceURI := range m.GetObjectCreated() {
			isWithinLimit, sizeErr := h.IsObjectSizeWithinLimit(ctx, sourceURI)
			if sizeErr != nil {
				logger.Error(sizeErr, "error checking object size")
				break
			}

			if !isWithinLimit {
				sizeErr = fmt.Errorf("%w: object size exceeds %.2f MB limit", ErrFileSizeLimitExceeded, float64(h.SizeLimit)/(1024*1024))
				logger.Error(sizeErr, "error copying file due to size limit")
				m.ErrorMessage = sizeErr.Error()
				response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
					ItemIdentifier: record.MessageId,
				})
				return response, sizeErr
			}

			copyInput := GetCopyObjectInput(sourceURI, h.DestinationURI)
			if _, cerr := h.S3Client.CopyObject(ctx, copyInput); cerr != nil {
				logger.Error(cerr, "error copying file yo!")
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
		LogPrefix:      cfg.LogPrefix,
		S3Client:       cfg.S3Client,
		Logger:         logr.Discard(),
		SizeLimit:      cfg.SizeLimit,
	}

	if cfg.Logger != nil {
		h.Logger = *cfg.Logger
	}

	return h, nil
}
