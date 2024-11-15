package metricsconfigurator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/observeinc/aws-sam-apps/pkg/logging"

	"net/http"

	"github.com/go-logr/logr"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const bearerTokenFormat = "Bearer %s %s"

type GraphQLRequest struct {
	Query string `json:"query"`
}

type Handler struct {
	Logger            logr.Logger
	MetricStreamName  string
	FirehoseArn       string
	RoleArn           string
	OutputFormat      string
	AccountID         string
	DatasourceID      string
	ObserveDomainName string
	SecretName        string
}

type Config struct {
	MetricStreamName  string `env:"METRIC_STREAM_NAME,required"`
	AccountID         string `env:"ACCOUNT_ID,required"`
	DatasourceID      string `env:"DATASOURCE_ID,required"`
	ObserveDomainName string `env:"OBSERVE_DOMAIN_NAME,required"`
	SecretName        string `env:"SECRET_NAME,required"`
	FirehoseArn       string `env:"FIREHOSE_ARN,required"`
	RoleArn           string `env:"ROLE_ARN,required"`
	OutputFormat      string `env:"OUTPUT_FORMAT,required"`
	Logging           *logging.Config
}

type MetricsListItem struct {
	Namespace   string   `json:"namespace"`
	MetricNames []string `json:"metricNames"`
}

type AwsCollectionStackConfig struct {
	AwsServiceMetricsList []MetricsListItem `json:"awsServiceMetricsList"`
}

type MetricsConfig struct {
	AwsCollectionStackConfig AwsCollectionStackConfig `json:"awsCollectionStackConfig"`
}

type GraphQLResponse struct {
	Data struct {
		Datasource struct {
			Name   string        `json:"name"`
			Config MetricsConfig `json:"config"`
		} `json:"datasource"`
	} `json:"data"`
}

func New(cfg *Config, logger logr.Logger) (Handler, error) {
	return Handler{
		Logger:            logger,
		MetricStreamName:  cfg.MetricStreamName,
		AccountID:         cfg.AccountID,
		SecretName:        cfg.SecretName,
		DatasourceID:      cfg.DatasourceID,
		ObserveDomainName: cfg.ObserveDomainName,
		FirehoseArn:       cfg.FirehoseArn,
		RoleArn:           cfg.RoleArn,
		OutputFormat:      cfg.OutputFormat,
	}, nil
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("handling request to configure metrics via lambda")

	// at this stage, we cannot report an error,
	// because the request with the response url is not parsed yet
	req, err := h.parsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.V(3).Info("parsed request", "request", *req)

	// Handle Delete case, directly delete
	if req.RequestType == "Delete" {
		logger.V(3).Info("delete request received, deleting lambda for metrics configuration")
		report_err := h.reportStatus(*req, true, "successfully deleted")
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation on successful delete: %w", report_err)
		}
		return []byte{}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, h.reportAndError("failed to load AWS config", req, err)
	}

	token, err := h.getSecretValue(ctx, cfg)
	if err != nil {
		return nil, h.reportAndError("failed to retrieve secret value", req, err)
	}
	logger.V(4).Info("retrieved token from secret manager")

	client := &http.Client{}
	bodyBytes, err := h.getDatasource(token, h.ObserveDomainName, client)
	if err != nil {
		return nil, h.reportAndError("failed to retrieve datasource", req, err)
	}
	logger.V(4).Info("retrieved datasource details")

	MetricsFilters, err := h.parseResponse(bodyBytes)
	if err != nil {
		return nil, h.reportAndError("failed to parse response", req, err)
	}
	logger.V(4).Info("parsed response, metric filters", "filters", MetricsFilters)

	// Create a cloudwatch client
	cwClient := cloudwatch.NewFromConfig(cfg)

	// add filters to provided metric stream
	_, err = cwClient.PutMetricStream(ctx, &cloudwatch.PutMetricStreamInput{
		FirehoseArn:    &h.FirehoseArn,
		RoleArn:        &h.RoleArn,
		OutputFormat:   types.MetricStreamOutputFormat(h.OutputFormat),
		Name:           &h.MetricStreamName,
		IncludeFilters: MetricsFilters,
	})
	if err != nil {
		return nil, h.reportAndError("failed to add filters to metric stream", req, err)
	}
	logger.V(4).Info("successfully added filters to metric stream")

	err = h.reportStatus(*req, true, "successfully wrote metrics to metric stream")
	if err != nil {
		return nil, fmt.Errorf("failed to report status to cloudformation: %w, during successful write", err)
	}

	logger.V(3).Info("returned response to cloudformation")
	return []byte{}, nil
}

func (h Handler) parsePayload(payload []byte) (*Request, error) {
	var req Request
	// we cannot report errors until we have parsed out the json for the request,
	// which contains the url to send the response to
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("failed to decode payload for metricsconfigurator: %w", err)
	}
	return &req, nil
}

func (h Handler) parseResponse(bodyBytes []byte) ([]types.MetricStreamFilter, error) {
	logger := h.Logger
	var result GraphQLResponse
	err := json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON from GQL response: %w", err)
	}

	logger.V(4).Info("response from observe api", "result", result)

	metricSelection := result.Data.Datasource.Config.AwsCollectionStackConfig.AwsServiceMetricsList

	MetricsFilters := convertToMetricStreamFilters(metricSelection)
	return MetricsFilters, nil
}

func convertToMetricStreamFilters(MetricsList []MetricsListItem) []types.MetricStreamFilter {
	// to ensure that an empty slice is returned instead of nil when `MetricsList` is empty
	metricsFilters := make([]types.MetricStreamFilter, 0)
	for _, service := range MetricsList {
		metricsFilters = append(metricsFilters, types.MetricStreamFilter{
			Namespace:   &service.Namespace,
			MetricNames: service.MetricNames,
		})
	}

	return metricsFilters
}

func (h Handler) getSecretValue(ctx context.Context, cfg aws.Config) (*string, error) {
	secretName := h.SecretName
	svc := secretsmanager.NewFromConfig(cfg)
	secretValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret value: %w", err)
	}
	token := *secretValue.SecretString
	return &token, nil
}

func (h Handler) getDatasource(token *string, observeDomainName string, client *http.Client) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("calling observe api", "AccountID", h.AccountID)

	fullToken := fmt.Sprintf(bearerTokenFormat, h.AccountID, *token)
	query := fmt.Sprintf(`
		{
			datasource(id: "%s") {
				name
				config {
					awsCollectionStackConfig {
						awsServiceMetricsList {
							namespace
							metricNames
						}
					}
				}
			}
		}
	`, h.DatasourceID)

	jsonData, err := json.Marshal(GraphQLRequest{Query: query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshall request into json: %w", err)
	}

	host := fmt.Sprintf("%s.%s", h.AccountID, observeDomainName)
	url := fmt.Sprintf("https://%s/v1/meta", host)

	request, reqErr := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fullToken)

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error receiving response from graphql: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return bodyBytes, nil
}

func (h Handler) reportAndError(reason string, request *Request, err error) error {
	// N.B. it is necessary to send a response to cloudformation
	// The lambda returning is not sufficient for cloudformation.
	reportErr := h.reportStatus(*request, false, reason)
	if reportErr != nil {
		return fmt.Errorf("failed to report status to cloudformation: %w, while reporting error, %s: %w", reportErr, reason, err)
	}
	return fmt.Errorf("%s: %w", reason, err)
}

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
		PhysicalResourceId: "lambda-metricsconfigurator",
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
