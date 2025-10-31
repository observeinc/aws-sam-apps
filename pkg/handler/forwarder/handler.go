package forwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-logr/logr"

	"github.com/observeinc/aws-sam-apps/pkg/handler"
)

var errNoLambdaContext = fmt.Errorf("no lambda context found")

type S3Client interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// UploaderAPI defines the interface for S3 Upload Manager
type UploaderAPI interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

// S3GetObjectAPI defines the interface for GetObject operations
type S3GetObjectAPI interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type Override interface {
	Apply(context.Context, *s3.CopyObjectInput) bool
}

type Handler struct {
	handler.Mux

	MaxFileSize        int64
	DestinationURI     *url.URL
	S3Client           S3Client
	Uploader           UploaderAPI // Upload Manager (automatically handles small and large files)
	Override           Override
	ObjectPolicy       interface{ Allow(string) bool }
	Now                func() time.Time
	MaxConcurrentTasks int
}

// GetCopyObjectInput constructs the input struct for CopyObject.
func GetCopyObjectInput(source, destination *url.URL) *s3.CopyObjectInput {
	if source == nil || destination == nil {
		return nil
	}

	var (
		bucket     = destination.Host
		copySource = fmt.Sprintf("%s%s", source.Host, source.Path)
		key        = source.Path
	)

	if destination.Scheme == "s3" {
		// empty string as base strips the leading slash
		key = fmt.Sprintf("%s%s", strings.Trim(destination.Path, "/"), source.Path)
	}

	key = strings.TrimLeft(key, "/")

	return &s3.CopyObjectInput{
		Bucket:     &bucket,
		CopySource: &copySource,
		Key:        &key,
	}
}

func (h *Handler) GetDestinationRegion(ctx context.Context, client s3.HeadBucketAPIClient) (string, error) {
	if h.DestinationURI.Scheme != "s3" {
		return "", nil
	}
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
		"AWSLogs",
		functionArn.AccountID,
		"sqs",
		functionArn.Region,
		now.Format("2006/01/02/15"), // use yyyy/mm/dd/hh format
		lctx.AwsRequestID,
	}, "/")

	if h.DestinationURI.Scheme == "s3" {
		key = strings.Trim(h.DestinationURI.Path, "/") + "/" + key
	}

	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(h.DestinationURI.Host),
		Key:         &key,
		Body:        r,
		ContentType: aws.String("application/x-aws-sqs"),
	}

	// Use Upload Manager for all uploads
	// For small files (< 5MB), it automatically uses single PutObject
	// For large files (> 5MB), it uses multipart upload with 10MB chunks
	if h.Uploader != nil {
		_, err = h.Uploader.Upload(ctx, putInput)
		if err != nil {
			return fmt.Errorf("failed to upload messages: %w", err)
		}
	} else {
		// Fallback to standard PutObject if Uploader not configured
		// This requires seekable stream for retries
		var body io.ReadSeeker
		if r != nil {
			data, err := io.ReadAll(r)
			if err != nil {
				return fmt.Errorf("failed to read message data: %w", err)
			}
			body = bytes.NewReader(data)
		}
		putInput.Body = body

		_, err = h.S3Client.PutObject(ctx, putInput)
		if err != nil {
			return fmt.Errorf("failed to write messages: %w", err)
		}
	}

	return nil
}

// copyObjectWithUploadManager performs a copy operation using GetObject + Upload Manager
func (h *Handler) copyObjectWithUploadManager(ctx context.Context, copyInput *s3.CopyObjectInput) error {
	logger := logr.FromContextOrDiscard(ctx)

	// Parse the CopySource to extract bucket and key
	parts := strings.SplitN(aws.ToString(copyInput.CopySource), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid copy source: %s", aws.ToString(copyInput.CopySource))
	}

	// Get the object from source
	getOutput, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(parts[0]),
		Key:    aws.String(parts[1]),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer func() {
		if closeErr := getOutput.Body.Close(); closeErr != nil {
			logger.V(4).Error(closeErr, "failed to close response body")
		}
	}()

	// Upload to destination using Upload Manager
	// Upload Manager automatically handles non-seekable streams
	logger.V(6).Info("using upload manager for copy", "source", copyInput.CopySource, "dest", copyInput.Key)

	// Build PutObjectInput, respecting any overrides set in copyInput
	putInput := &s3.PutObjectInput{
		Bucket:      copyInput.Bucket,
		Key:         copyInput.Key,
		Body:        getOutput.Body,
		ContentType: getOutput.ContentType,
		Metadata:    getOutput.Metadata,
	}

	// Respect content type override if set
	if ct := copyInput.ContentType; ct != nil {
		putInput.ContentType = ct
	}

	// Respect content encoding override if set
	if ce := copyInput.ContentEncoding; ce != nil {
		putInput.ContentEncoding = ce
	}

	_, err = h.Uploader.Upload(ctx, putInput)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

func (h *Handler) ProcessRecord(ctx context.Context, record *events.SQSMessage) error {
	logger := logr.FromContextOrDiscard(ctx)

	copyRecords := GetObjectCreated(record)
	for _, copyRecord := range copyRecords {
		sourceURL, err := url.Parse(copyRecord.URI)
		if err != nil {
			logger.Error(err, "error parsing source URI", "SourceURI", copyRecord.URI)
			continue
		}

		copyInput := GetCopyObjectInput(sourceURL, h.DestinationURI)

		if !h.ObjectPolicy.Allow(aws.ToString(copyInput.CopySource)) {
			logger.Info("Ignoring object not in allowed sources", "bucket", copyInput.Bucket, "key", copyInput.Key)
			continue
		}

		if copyRecord.Size != nil && h.MaxFileSize > 0 && *copyRecord.Size > h.MaxFileSize {
			logger.V(1).Info("object size exceeds the maximum file size limit; skipping copy",
				"max", h.MaxFileSize, "size", *copyRecord.Size, "uri", copyRecord.URI)
			// Log a warning and skip this object by continuing to the next iteration
			continue
		}

		if h.Override != nil {
			if h.Override.Apply(ctx, copyInput) && copyInput.Key == nil {
				logger.V(6).Info("ignoring object")
				continue
			}
		}

		// Use Upload Manager for all S3 copy operations
		// For small files (< 5MB), it automatically uses single PutObject
		// For large files (> 5MB), it uses multipart upload
		if h.Uploader != nil {
			err = h.copyObjectWithUploadManager(ctx, copyInput)
		} else {
			// Fallback to standard CopyObject if Uploader not configured
			_, err = h.S3Client.CopyObject(ctx, copyInput)
		}

		if err != nil {
			return fmt.Errorf("error copying file %q: %w", copyRecord.URI, err)
		}
	}
	return nil
}

func (h *Handler) Handle(ctx context.Context, request events.SQSEvent) (response events.SQSEventResponse, err error) {
	logger := logr.FromContextOrDiscard(ctx)

	resultCh := make(chan *SQSMessage, len(request.Records))
	defer close(resultCh)

	var (
		acquireToken = func() {}
		releaseToken = func() {}
	)

	if h.MaxConcurrentTasks > 0 {
		limitCh := make(chan struct{}, h.MaxConcurrentTasks)
		defer close(limitCh)
		acquireToken = func() { limitCh <- struct{}{} }
		releaseToken = func() { <-limitCh }
	}

	for _, record := range request.Records {
		acquireToken()
		go func(m events.SQSMessage) {
			defer releaseToken()
			result := &SQSMessage{SQSMessage: m}
			if err := h.ProcessRecord(ctx, &m); err != nil {
				logger.Error(err, "failed to process record")
				result.ErrorMessage = err.Error()
			}
			resultCh <- result
		}(record)
	}

	var messages bytes.Buffer
	encoder := json.NewEncoder(&messages)
	for i := 0; i < len(request.Records); i++ {
		result := <-resultCh
		if result.ErrorMessage != "" {
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: result.MessageId,
			})
		}
		if e := encoder.Encode(result); e != nil {
			err = errors.Join(err, fmt.Errorf("failed to encode message: %w", e))
		}
	}

	if err == nil {
		err = h.WriteSQS(ctx, &messages)
	}

	return
}

func New(cfg *Config) (h *Handler, err error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	u, _ := url.ParseRequestURI(cfg.DestinationURI)

	maxConcurrentTasks := cfg.MaxConcurrentTasks
	if maxConcurrentTasks == 0 {
		maxConcurrentTasks = runtime.NumCPU()
	}

	objectFilter, _ := NewObjectFilter(cfg.SourceBucketNames, cfg.SourceObjectKeys)

	h = &Handler{
		DestinationURI:     u,
		S3Client:           cfg.S3Client,
		Uploader:           cfg.Uploader,
		MaxFileSize:        cfg.MaxFileSize,
		Override:           cfg.Override,
		ObjectPolicy:       objectFilter,
		Now:                time.Now,
		MaxConcurrentTasks: maxConcurrentTasks,
	}

	return h, nil
}
