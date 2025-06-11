package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/pkg/profile"

	"github.com/observeinc/aws-sam-apps/pkg/lambda"
	"github.com/observeinc/aws-sam-apps/pkg/lambda/forwarder"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
	"github.com/observeinc/aws-sam-apps/pkg/testing/awstest"
)

func realInit() error {
	var (
		contentType     = flag.String("content-type", "", "Force content type")
		contentEncoding = flag.String("content-encoding", "", "Force content encoding")
		outputDir       = flag.String("output", "forwarder-results", "Output directory where request body and profiling info are dumped")
		profileMode     = flag.String("profile", "", "enable profiling mode, one of [cpu, mem, mutex, block]")
	)

	flag.Parse()

	inputFiles := flag.Args()
	if len(inputFiles) == 0 {
		return nil
	}

	// create output directory
	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// setup profiling
	{
		options := []func(*profile.Profile){profile.ProfilePath(*outputDir)}
		switch *profileMode {
		case "cpu":
			options = append(options, profile.CPUProfile)
		case "mem":
			options = append(options, profile.MemProfile)
		case "mutex":
			options = append(options, profile.MutexProfile)
		case "block":
			options = append(options, profile.BlockProfile)
		}
		defer profile.Start(options...).Stop()
	}

	// setup fake HTTP server, which dumps request bodies to output directory
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.CreateTemp("", "forwarder")
		if err != nil {
			w.WriteHeader(400)
		}
		// remove file if it still exists by the end of processing
		defer func() {
			if err := os.Remove(f.Name()); err != nil {
				log.Printf("Failed to remove file %s: %v", f.Name(), err)
			}
		}()

		hasher := sha256.New()
		body := io.TeeReader(r.Body, hasher)
		if _, err := io.Copy(f, body); err != nil {
			w.WriteHeader(400)
		}

		hash := hex.EncodeToString(hasher.Sum(nil))[:8]
		err = os.Rename(f.Name(), filepath.Join(*outputDir, fmt.Sprintf("%s.ndjson.gz", hash)))
		if err != nil {
			w.WriteHeader(400)
		}
	}))
	defer s.Close()

	config := forwarder.Config{
		DestinationURI:         s.URL,
		SourceBucketNames:      []string{"*"},
		HTTPInsecureSkipVerify: true,
		Logging: &logging.Config{
			Verbosity:   9,
			AddSource:   false,
			HandlerType: "text",
		},
		AWSS3Client: &awstest.FileGetter{
			ContentType:     contentType,
			ContentEncoding: contentEncoding,
		},
	}

	ctx := context.Background()
	// instantiate struct in order to inherit defaults
	if err := lambda.ProcessEnv(ctx, &config); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// create forwarder
	fwd, err := forwarder.New(ctx, &config)
	if err != nil {
		return fmt.Errorf("failed to configure entrypoint: %w", err)
	}

	var records []events.SQSMessage
	for _, inputFile := range inputFiles {
		records = append(records, events.SQSMessage{
			Body: fmt.Sprintf(`{"copy": [{"uri": "s3://%s"}]}`, inputFile),
		})
	}

	event, err := json.Marshal(events.SQSEvent{Records: records})
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	ctx = lambdacontext.NewContext(ctx, &lambdacontext.LambdaContext{
		AwsRequestID:       "00000000-0000-0000-0000-000000000000",
		InvokedFunctionArn: "arn:aws:lambda:us-west-2:123456789012:function:my-function",
	})

	// process event
	resp, err := fwd.Entrypoint.Invoke(ctx, event)
	if err != nil {
		return fmt.Errorf("invocation failed: %w", err)
	}

	var eventResponse events.SQSEventResponse
	if err := json.Unmarshal(resp, &eventResponse); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}
	if len(eventResponse.BatchItemFailures) > 0 {
		return fmt.Errorf("batch item failures")
	}
	return nil
}

func main() {
	if err := realInit(); err != nil {
		panic(err)
	}
}
