package metricsrecorder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/observeinc/aws-sam-apps/pkg/logging"

	"net/http"

	"github.com/go-logr/logr"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type GraphQLRequest struct {
	Query string `json:"query"`
}

type Handler struct {
	Logger           logr.Logger
	metricStreamName string
	BearerToken      string
	FirehoseArn      string
	RoleArn          string
	OutputFormat     string
	AccountNumber    int
}

type Config struct {
	MetricStreamName string `env:"METRIC_STREAM_NAME,required"`
	BearerToken      string `env:"BEARER_TOKEN,required"`
	AccountNumber    int    `env:"ACCOUNT_NUMBER,required"`
	FirehoseArn      string `env:"FIREHOSE_ARN,required"`
	RoleArn          string `env:"ROLE_ARN,required"`
	OutputFormat     string `env:"OUTPUT_FORMAT,required"`
	Logging          *logging.Config
}

type MetricsListItem struct {
	Namespace   string   `json:"namespace"`
	MetricNames []string `json:"metricNames"`
}

func convertToMetricStreamFilters(MetricsList []MetricsListItem) []types.MetricStreamFilter {

	var metricsFilters []types.MetricStreamFilter

	for _, service := range MetricsList {
		metricsFilters = append(metricsFilters, types.MetricStreamFilter{
			Namespace:   &service.Namespace,
			MetricNames: service.MetricNames,
		})
	}

	return metricsFilters
}

type GraphQLResponse struct {
	Data struct {
		Datasource struct {
			Name      string `json:"name"`
			Variables []struct {
				Details struct {
					AwsCollectionStackConfig struct {
						AwsServiceMetricsList []MetricsListItem `json:"awsServiceMetricsList"`
					} `json:"awsCollectionStackConfig"`
				} `json:"details"`
			} `json:"variables"`
		} `json:"datasource"`
	} `json:"data"`
}

func New(cfg *Config, logger logr.Logger) (Handler, error) {
	return Handler{
		Logger:           logger,
		metricStreamName: cfg.MetricStreamName,
		BearerToken:      cfg.BearerToken,
		AccountNumber:    cfg.AccountNumber,
		FirehoseArn:      cfg.FirehoseArn,
		RoleArn:          cfg.RoleArn,
		OutputFormat:     cfg.OutputFormat,
	}, nil
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("handling request")

	// unparse payload
	dec := json.NewDecoder(bytes.NewReader(payload))

	var rawResult map[string]interface{}

	// Unmarshal the byte array into the result variable
	err := json.Unmarshal(payload, &rawResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	logger.V(4).Info("handling request")
	logger.Info("Parsed JSON", "data", rawResult)

	var req Request

	// we cannot return errors until we have parsed out the json for the request,
	// which contains the url to send the response to
	if err := dec.Decode(&req); err != nil {

		// make best effort to send response to cloudformation
		h.reportStatus(req, false, "failed to decode payload for metricsrecorder")
		return nil, fmt.Errorf("failed to decode payload for metricsrecorder: %w", err)
	}

	if req.RequestType == "Delete" {
		// if delete, do nothing, allow to be deleted
		h.reportStatus(req, true, "successfully deleted")
		return []byte{}, nil
	}

	metricStreamName := h.metricStreamName

	// get necessary info for api call (tbd what this is exactly, acct no + gql token)
	// call observe api to get metrics settings from postgres
	token := h.BearerToken
	accountNumber := h.AccountNumber
	logger.V(4).Info("calling observe api", "token", token)
	logger.V(4).Info("calling observe api", "accountNumber", accountNumber)

	fullToken := fmt.Sprintf("Bearer %d %s", accountNumber, token)

	query := `
		{
			datasource(id: "41752136") {
				name
				variables {
					details {
						awsCollectionStackConfig {
							awsServiceMetricsList {
								namespace
								metricNames
							}
						}
					}
				}
			}
		}
	`

	gqlRequest := GraphQLRequest{Query: query}
	jsonData, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request into gql: %w", err)
	}

	host := fmt.Sprintf("%d.observe-eng.com", accountNumber)
	url := fmt.Sprintf("https://%s/v1/meta", host)

	request, reqErr := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fullToken)

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error receiving response from graphql: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var result GraphQLResponse
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON from GQL response: %w", err)
	}

	var metricSelection []MetricsListItem

	if len(result.Data.Datasource.Variables) > 0 {
		metricSelection = result.Data.Datasource.Variables[0].Details.AwsCollectionStackConfig.AwsServiceMetricsList
	}

	MetricsFilters := convertToMetricStreamFilters(metricSelection)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		h.reportStatus(req, false, "failed to load AWS config")
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create a cloudwatch client
	cwClient := cloudwatch.NewFromConfig(cfg)

	// add filters to provided metric stream
	_, err = cwClient.PutMetricStream(ctx, &cloudwatch.PutMetricStreamInput{
		FirehoseArn:    &h.FirehoseArn,
		RoleArn:        &h.RoleArn,
		OutputFormat:   types.MetricStreamOutputFormat(h.OutputFormat),
		Name:           &metricStreamName,
		IncludeFilters: MetricsFilters,
	})
	if err != nil {
		h.reportStatus(req, false, fmt.Sprintf("failed to add filters to metric stream: %s", err))
		return nil, fmt.Errorf("failed to add filters to metric stream: %w", err)
	}

	logger.V(4).Info("successfully wrote metrics to s3")
	err = h.reportStatus(req, true, "successfully wrote metrics to s3")
	if err != nil {
		return nil, fmt.Errorf("failed to report status to cloudformation: %w", err)
	}

	logger.V(4).Info("returned response to cloudformation")
	return []byte{}, nil
}

// N.B. it is necessary to send a response to cloudformation
// The lambda returning is not sufficient for cloudformation.
func (h Handler) reportStatus(request Request, success bool, reason string) error {
	logger := h.Logger
	var statusString string

	if success {
		statusString = "SUCCESS"
	} else {
		statusString = "FAILED"
	}

	resp := CfResponse{
		Status:             statusString,
		PhysicalResourceId: "lambda-metricsrecorder",
		Reason:             reason,
		StackId:            request.StackId,
		RequestId:          request.RequestId,
		LogicalResourceId:  request.LogicalResourceId,
	}

	// send response to cloudformation
	body, _ := json.Marshal(resp)

	logger.V(4).Info("reporting status to cloudformation", "response", resp)
	req, _ := http.NewRequest("PUT", request.ResponseURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	logger.V(4).Info("request created", "url", request.ResponseURL)

	client := &http.Client{}
	_, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send response to cloudformation: %w", err)
	}

	return nil
}
