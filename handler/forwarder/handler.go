package forwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/handler"
)

var errNoLambdaContext = fmt.Errorf("no lambda context found")

type S3Client interface {
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Override interface {
	Apply(context.Context, *s3.CopyObjectInput) bool
}

type Handler struct {
	handler.Mux
	MaxFileSize       int64
	DestinationURI    *url.URL
	S3Client          S3Client
	Override          Override
	SourceBucketNames []string
	Now               func() time.Time
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

func (h *Handler) GetDestinationRegion(ctx context.Context, client s3.HeadBucketAPIClient) (string, error) {
	region, err := manager.GetBucketRegion(ctx, client, h.DestinationURI.Host)
	if err != nil {
		return "", fmt.Errorf("failed to get region: %w", err)
	}
	return region, nil
}

func (h *Handler) WriteSQS(ctx context.Context, r io.Reader) error {
	lctx, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return errNoLambdaContext
	}

	functionArn, err := arn.Parse(lctx.InvokedFunctionArn)
	if err != nil {
		return fmt.Errorf("failed to parse function ARN: %w", err)
	}

	now := h.Now()
	key := strings.Join([]string{
		strings.Trim(h.DestinationURI.Path, "/"),
		"AWSLogs",
		functionArn.AccountID,
		"sqs",
		functionArn.Region,
		now.Format("2006/01/02/15"), // use yyyy/mm/dd/hh format
		lctx.AwsRequestID,
	}, "/")

	_, err = h.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(h.DestinationURI.Host),
		Key:         &key,
		Body:        r,
		ContentType: aws.String("application/x-aws-sqs"),
	})
	if err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}
	return nil
}

func (h *Handler) Handle(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	logger := logr.FromContextOrDiscard(ctx)

	var messages bytes.Buffer
	defer func() {
		if err == nil {
			logger.V(3).Info("logging messages")
			err = h.WriteSQS(ctx, &messages)
		}
	}()

	encoder := json.NewEncoder(&messages)

	for _, record := range request.Records {
		m := &SQSMessage{SQSMessage: record}
		copyRecords := m.GetObjectCreated()
		for _, copyRecord := range copyRecords {
			sourceURL, err := url.Parse(copyRecord.URI)
			if err != nil {
				logger.Error(err, "error parsing source URI", "SourceURI", copyRecord.URI)
				continue
			}

			if !h.isBucketAllowed(sourceURL.Host) {
				logger.Info("Received event from a bucket not in the allowed list; skipping", "bucket", sourceURL.Host)
				continue
			}
			if copyRecord.Size != nil && h.MaxFileSize > 0 && *copyRecord.Size > h.MaxFileSize {
				logger.V(1).Info("object size exceeds the maximum file size limit; skipping copy",
					"max", h.MaxFileSize, "size", *copyRecord.Size, "uri", copyRecord.URI)
				// Log a warning and skip this object by continuing to the next iteration
				continue
			}

			copyInput := GetCopyObjectInput(sourceURL, h.DestinationURI)

			if h.Override != nil {
				if h.Override.Apply(ctx, copyInput) && copyInput.Key == nil {
					logger.V(6).Info("ignoring object")
					continue
				}
			}

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

// isBucketAllowed checks if the given bucket is in the allowed list or matches a pattern.
func (h *Handler) isBucketAllowed(bucket string) bool {
	for _, pattern := range h.SourceBucketNames {
		if match, _ := filepath.Match(pattern, bucket); match {
			return true
		}
	}
	return false
}

func New(cfg *Config) (h *Handler, err error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	u, _ := url.ParseRequestURI(cfg.DestinationURI)

	h = &Handler{
		DestinationURI:    u,
		S3Client:          cfg.S3Client,
		MaxFileSize:       cfg.MaxFileSize,
		Override:          cfg.Override,
		SourceBucketNames: cfg.SourceBucketNames,
		Now:               time.Now,
	}

	return h, nil
}
