package metricsconfigurer

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

type GraphQLRequest struct {
	Query string `json:"query"`
}

type Handler struct {
	Logger           logr.Logger
	MetricStreamName string
	FirehoseArn      string
	RoleArn          string
	OutputFormat     string
	AccountNumber    string
	DatasourceID     string
	DomainName       string
	SecretName       string
}

type Config struct {
	MetricStreamName string `env:"METRIC_STREAM_NAME,required"`
	AccountNumber    string `env:"ACCOUNT_NUMBER,required"`
	DatasourceID     string `env:"DATASOURCE_ID,required"`
	DomainName       string `env:"DOMAIN_NAME,required"`
	SecretName       string `env:"SECRET_NAME,required"`
	FirehoseArn      string `env:"FIREHOSE_ARN,required"`
	RoleArn          string `env:"ROLE_ARN,required"`
	OutputFormat     string `env:"OUTPUT_FORMAT,required"`
	Logging          *logging.Config
}

type MetricsListItem struct {
	Namespace   string   `json:"namespace"`
	MetricNames []string `json:"metricNames"`
}

type AwsCollectionStackConfig struct {
	AwsServiceMetricsList []MetricsListItem `json:"awsServiceMetricsList"`
}

type Details struct {
	AwsCollectionStackConfig AwsCollectionStackConfig `json:"awsCollectionStackConfig"`
}

type Variable struct {
	Name    string  `json:"name"`
	Details Details `json:"details"`
}

type GraphQLResponse struct {
	Data struct {
		Datasource struct {
			Name      string     `json:"name"`
			Variables []Variable `json:"variables"`
		} `json:"datasource"`
	} `json:"data"`
}

func New(cfg *Config, logger logr.Logger) (Handler, error) {
	return Handler{
		Logger:           logger,
		MetricStreamName: cfg.MetricStreamName,
		AccountNumber:    cfg.AccountNumber,
		SecretName:       cfg.SecretName,
		DatasourceID:     cfg.DatasourceID,
		DomainName:       cfg.DomainName,
		FirehoseArn:      cfg.FirehoseArn,
		RoleArn:          cfg.RoleArn,
		OutputFormat:     cfg.OutputFormat,
	}, nil
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("handling request")

	req, err := h.parsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Handle Delete case, directly delete
	if req.RequestType == "Delete" {
		report_err := h.reportStatus(*req, true, "successfully deleted")
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation on successful delete: %w", report_err)
		}
		return []byte{}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		report_err := h.reportStatus(*req, false, "failed to load AWS config")
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	token, err := h.getSecretValue(ctx, req, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret value: %w", err)
	}

	client := &http.Client{}
	bodyBytes, err := h.getDatasource(req, token, h.DomainName, client)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve datasource: %w", err)
	}

	MetricsFilters, err := h.parseResponse(bodyBytes, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

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
		report_err := h.reportStatus(*req, false, fmt.Sprintf("failed to add filters to metric stream: %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to add filters to metric stream: %w", err)
	}

	logger.V(4).Info("successfully wrote metrics to metric stream")
	err = h.reportStatus(*req, true, "successfully wrote metrics to metric stream")
	if err != nil {
		return nil, fmt.Errorf("failed to report status to cloudformation: %w, during successful write", err)
	}

	logger.V(4).Info("returned response to cloudformation")
	return []byte{}, nil
}

func (h Handler) parsePayload(payload []byte) (*Request, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))

	var rawResult map[string]interface{}

	// Unmarshal the byte array into the result variable
	err := json.Unmarshal(payload, &rawResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}
	var req Request

	// we cannot return errors until we have parsed out the json for the request,
	// which contains the url to send the response to
	if err := dec.Decode(&req); err != nil {
		// make best effort to send response to cloudformation
		report_err := h.reportStatus(req, false, "failed to decode payload for metricsconfigurer")
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to decode payload for metricsconfigurer: %w", err)
	}
	return &req, nil
}

func (h Handler) parseResponse(bodyBytes []byte, req *Request) ([]types.MetricStreamFilter, error) {
	logger := h.Logger
	var result GraphQLResponse
	err := json.Unmarshal(bodyBytes, &result)
	if err != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("error unmarshalling JSON from GQL response %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("error unmarshalling JSON from GQL response: %w", err)
	}

	logger.V(4).Info("response from observe api", "result", result)

	var metricSelection []MetricsListItem

	variablesList := result.Data.Datasource.Variables

	var targetVariable *Variable

	// Iterate through the variables and find one with the name "Metrics"
	// This is where we should find the metrics configuration
	for _, variable := range variablesList {
		if variable.Name == "Metrics" {
			targetVariable = &variable
			break
		}
	}

	if targetVariable == nil {
		report_err := h.reportStatus(*req, false, "metrics variable not set in datasource")
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("metrics variable not set in datasource, %+v", result)
	}

	metricSelection = targetVariable.Details.AwsCollectionStackConfig.AwsServiceMetricsList

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

func (h Handler) getSecretValue(ctx context.Context, req *Request, cfg aws.Config) (*string, error) {

	secretName := h.SecretName
	svc := secretsmanager.NewFromConfig(cfg)
	secretValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("failed to retrieve secret value %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to retrieve secret value: %w", err)
	}

	token := *secretValue.SecretString

	return &token, nil
}

func (h Handler) getDatasource(req *Request, token *string, domainName string, client *http.Client) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("calling observe api", "accountNumber", h.AccountNumber)

	fullToken := fmt.Sprintf("Bearer %s %s", h.AccountNumber, *token)
	query := fmt.Sprintf(`
		{
			datasource(id: "%s") {
				name
				variables {
					name
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
	`, h.DatasourceID)

	jsonData, err := json.Marshal(GraphQLRequest{Query: query})
	if err != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("failed to marshall request into json %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to marshall request into json: %w", err)
	}

	host := fmt.Sprintf("%s.%s", h.AccountNumber, domainName)
	url := fmt.Sprintf("https://%s/v1/meta", host)

	request, reqErr := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if reqErr != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("failed to create request %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fullToken)

	resp, err := client.Do(request)
	if err != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("error receiving response from graphql %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("error receiving response from graphql: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		report_err := h.reportStatus(*req, false, fmt.Sprintf("error reading response body %s", err))
		if report_err != nil {
			return nil, fmt.Errorf("failed to report status to cloudformation: %w, for error %w", report_err, err)
		}
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return bodyBytes, nil
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
		PhysicalResourceId: "lambda-metricsconfigurer",
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
