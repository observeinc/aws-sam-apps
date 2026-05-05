package metricsconfigurator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-logr/logr"
	"github.com/observeinc/aws-sam-apps/pkg/logging"
)

const bearerTokenFormat = "Bearer %s %s"
const maxNumMetricNames = 1000

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
	FilterUri         string
}

type Config struct {
	MetricStreamName  string `env:"METRIC_STREAM_NAME,required"`
	FirehoseArn       string `env:"FIREHOSE_ARN,required"`
	RoleArn           string `env:"ROLE_ARN,required"`
	OutputFormat      string `env:"OUTPUT_FORMAT,required"`
	AccountID         string `env:"ACCOUNT_ID"`
	DatasourceID      string `env:"DATASOURCE_ID"`
	ObserveDomainName string `env:"OBSERVE_DOMAIN_NAME"`
	SecretName        string `env:"SECRET_NAME"`
	FilterUri         string `env:"FILTER_URI"`
	Logging           *logging.Config
}

type MetricsListItem struct {
	Namespace   string   `json:"namespace"`
	MetricNames []string `json:"metricNames"`
}

type AwsCollectionStackConfig struct {
	AwsServiceMetricsList []MetricsListItem `json:"awsServiceMetricsList"`
	CustomMetricsList     []MetricsListItem `json:"customMetricsList"`
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
	if cfg.DatasourceID == "" && cfg.FilterUri == "" {
		return Handler{}, fmt.Errorf("either DATASOURCE_ID or FILTER_URI must be set")
	}
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
		FilterUri:         cfg.FilterUri,
	}, nil
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("handling request to configure metrics via lambda")

	req, err := h.parsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.V(3).Info("parsed request", "request", *req)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		if req.RequestType == "Delete" {
			logger.Error(err, "failed to load AWS config during delete, skipping metric stream cleanup")
			if reportErr := h.reportStatus(*req, true, "deleted (skipped metric stream cleanup: could not load AWS config)"); reportErr != nil {
				return nil, fmt.Errorf("failed to report status to cloudformation: %w", reportErr)
			}
			return []byte{}, nil
		}
		return nil, h.reportAndError("failed to load AWS config", req, err)
	}

	if req.RequestType == "Delete" {
		return h.handleDelete(ctx, cfg, req)
	}

	if h.DatasourceID != "" {
		err = h.invokeDatasourcePath(ctx, cfg, req)
	} else {
		err = h.invokeFilterUriPath(ctx, cfg, req)
	}
	if err != nil {
		return nil, err
	}

	logger.V(3).Info("returned response to cloudformation")
	return []byte{}, nil
}

func (h Handler) invokeDatasourcePath(ctx context.Context, cfg aws.Config, req *Request) error {
	logger := h.Logger

	token, err := h.getSecretValue(ctx, cfg)
	if err != nil {
		return h.reportAndError("failed to retrieve secret value", req, err)
	}
	logger.V(4).Info("retrieved token from secret manager")

	client := &http.Client{}
	bodyBytes, err := h.getDatasource(token, h.ObserveDomainName, client)
	if err != nil {
		return h.reportAndError("failed to retrieve datasource", req, err)
	}
	logger.V(4).Info("retrieved datasource details")

	metricsFilters, err := h.parseResponse(bodyBytes)
	if err != nil {
		return h.reportAndError("failed to parse response", req, err)
	}
	logger.V(4).Info("parsed response, metric filters", "filters", metricsFilters)

	cwClient := cloudwatch.NewFromConfig(cfg)

	filterGroups := h.makeMetricGroups(metricsFilters)
	for idx, filterGroup := range filterGroups {
		name := fmt.Sprintf("%s-%s-%d", h.MetricStreamName, "metric-stream", idx)
		_, err = cwClient.PutMetricStream(ctx, &cloudwatch.PutMetricStreamInput{
			FirehoseArn:    &h.FirehoseArn,
			RoleArn:        &h.RoleArn,
			OutputFormat:   types.MetricStreamOutputFormat(h.OutputFormat),
			Name:           &name,
			IncludeFilters: filterGroup,
		})
		if err != nil {
			return h.reportAndError("failed to add filter to metric stream", req, err)
		}
	}

	logger.V(4).Info("successfully added all filters to metric stream")
	err = h.reportStatus(*req, true, "successfully wrote metrics to metric stream")
	if err != nil {
		return fmt.Errorf("failed to report status to cloudformation: %w, during successful write", err)
	}
	return nil
}

func (h Handler) invokeFilterUriPath(ctx context.Context, cfg aws.Config, req *Request) error {
	logger := h.Logger
	logger.V(4).Info("using FilterUri path", "filterUri", h.FilterUri)

	data, err := downloadFilterYAML(ctx, h.FilterUri)
	if err != nil {
		return h.reportAndError("failed to download filter YAML", req, err)
	}
	logger.V(4).Info("downloaded filter YAML", "size", len(data))

	parsed, err := parseFilterYAML(data)
	if err != nil {
		return h.reportAndError("failed to parse filter YAML", req, err)
	}

	cwClient := cloudwatch.NewFromConfig(cfg)
	name := fmt.Sprintf("%s-%s-%d", h.MetricStreamName, "metric-stream", 0)

	input := &cloudwatch.PutMetricStreamInput{
		FirehoseArn:  &h.FirehoseArn,
		RoleArn:      &h.RoleArn,
		OutputFormat: types.MetricStreamOutputFormat(h.OutputFormat),
		Name:         &name,
	}

	if len(parsed.IncludeFilters) > 0 {
		filterGroups := h.makeMetricGroups(parsed.IncludeFilters)
		for idx, filterGroup := range filterGroups {
			streamName := fmt.Sprintf("%s-%s-%d", h.MetricStreamName, "metric-stream", idx)
			_, err = cwClient.PutMetricStream(ctx, &cloudwatch.PutMetricStreamInput{
				FirehoseArn:    &h.FirehoseArn,
				RoleArn:        &h.RoleArn,
				OutputFormat:   types.MetricStreamOutputFormat(h.OutputFormat),
				Name:           &streamName,
				IncludeFilters: filterGroup,
			})
			if err != nil {
				return h.reportAndError("failed to put metric stream with include filters", req, err)
			}
		}
	} else if len(parsed.ExcludeFilters) > 0 {
		input.ExcludeFilters = parsed.ExcludeFilters
		_, err = cwClient.PutMetricStream(ctx, input)
		if err != nil {
			return h.reportAndError("failed to put metric stream with exclude filters", req, err)
		}
	}

	logger.V(4).Info("successfully configured metric stream from FilterUri")
	err = h.reportStatus(*req, true, "successfully configured metric stream from FilterUri")
	if err != nil {
		return fmt.Errorf("failed to report status to cloudformation: %w, during successful write", err)
	}
	return nil
}

func (h Handler) handleDelete(ctx context.Context, cfg aws.Config, req *Request) ([]byte, error) {
	logger := h.Logger
	logger.V(3).Info("delete request received, cleaning up metric streams")

	cwClient := cloudwatch.NewFromConfig(cfg)

	var deleted int
	for idx := 0; ; idx++ {
		name := fmt.Sprintf("%s-metric-stream-%d", h.MetricStreamName, idx)
		_, err := cwClient.DeleteMetricStream(ctx, &cloudwatch.DeleteMetricStreamInput{
			Name: &name,
		})
		if err != nil {
			var rnf *types.ResourceNotFoundException
			if errors.As(err, &rnf) {
				break
			}
			logger.Error(err, "failed to delete metric stream, continuing", "name", name)
			break
		}
		deleted++
		logger.V(3).Info("deleted metric stream", "name", name)
	}

	reason := fmt.Sprintf("successfully deleted %d metric stream(s)", deleted)
	if reportErr := h.reportStatus(*req, true, reason); reportErr != nil {
		return nil, fmt.Errorf("failed to report status to cloudformation: %w", reportErr)
	}
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

	awsServiceMetrics := result.Data.Datasource.Config.AwsCollectionStackConfig.AwsServiceMetricsList
	customMetrics := result.Data.Datasource.Config.AwsCollectionStackConfig.CustomMetricsList

	// Combine both lists
	allMetrics := append(awsServiceMetrics, customMetrics...)

	MetricsFilters := convertToMetricStreamFilters(allMetrics)
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
						customMetricsList {
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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			logger.Error(closeErr, "failed to close response body")
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return bodyBytes, nil
}

func (h Handler) makeMetricGroups(MetricsFilters []types.MetricStreamFilter) [][]types.MetricStreamFilter {
	// create appropriate number of metric streams for metrics, assign metrics to streams

	currentNameCount := 0
	filterGroups := make([][]types.MetricStreamFilter, 0)
	currentFilterGroup := make([]types.MetricStreamFilter, 0)
	for _, filter := range MetricsFilters {
		currentNameCount += len(filter.MetricNames) + 1
		if currentNameCount > maxNumMetricNames {
			filterGroups = append(filterGroups, currentFilterGroup)
			currentFilterGroup = make([]types.MetricStreamFilter, 0)
			currentNameCount = len(filter.MetricNames) + 1
		}
		currentFilterGroup = append(currentFilterGroup, filter)
	}

	filterGroups = append(filterGroups, currentFilterGroup)
	return filterGroups
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
