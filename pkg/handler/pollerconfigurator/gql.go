package pollerconfigurator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
)

const bearerTokenFormat = "Bearer %s %s"

const createPollerMutation = `mutation CreatePoller($workspaceId: ObjectId!, $poller: PollerInput!) {
	createPoller(workspaceId: $workspaceId, poller: $poller) { id name }
}`

const updatePollerMutation = `mutation UpdatePoller($id: ObjectId!, $poller: PollerInput!) {
	updatePoller(id: $id, poller: $poller) { id name }
}`

const deletePollerMutation = `mutation DeletePoller($id: ObjectId!) {
	deletePoller(id: $id) { success }
}`

type graphQLRequest struct {
	Query     string      `json:"query"`
	Variables interface{} `json:"variables,omitempty"`
}

type createPollerResponse struct {
	Data struct {
		CreatePoller struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"createPoller"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type updatePollerResponse struct {
	Data struct {
		UpdatePoller struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"updatePoller"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type deletePollerResponse struct {
	Data struct {
		DeletePoller struct {
			Success bool `json:"success"`
		} `json:"deletePoller"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// GQL variable input types — serialized as JSON and sent in the
// "variables" field of the GraphQL request. Field names and string
// encoding (e.g. period as "300") match the Observe v1/meta schema.

type pollerInput struct {
	Name                    string                `json:"name,omitempty"`
	DatastreamId            string                `json:"datastreamId,omitempty"`
	Interval                string                `json:"interval,omitempty"`
	Retries                 *string               `json:"retries,omitempty"`
	CloudWatchMetricsConfig *cwMetricsConfigInput `json:"cloudWatchMetricsConfig"`
}

type cwMetricsConfigInput struct {
	Period        string             `json:"period"`
	Delay         string             `json:"delay"`
	Region        string             `json:"region"`
	AssumeRoleArn string             `json:"assumeRoleArn"`
	Queries       []queryVarInput    `json:"queries"`
}

type queryVarInput struct {
	Namespace      string                `json:"namespace"`
	MetricNames    []string              `json:"metricNames,omitempty"`
	Dimensions     []dimensionVarInput   `json:"dimensions,omitempty"`
	ResourceFilter *resourceFilterInput  `json:"resourceFilter,omitempty"`
}

type dimensionVarInput struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type resourceFilterInput struct {
	ResourceType  string            `json:"resourceType,omitempty"`
	Pattern       string            `json:"pattern,omitempty"`
	DimensionName string            `json:"dimensionName,omitempty"`
	TagFilters    []tagFilterInput  `json:"tagFilters"`
}

type tagFilterInput struct {
	Key    string   `json:"key"`
	Values []string `json:"values,omitempty"`
}

func buildPollerInput(cfg *PollerConfig, region, assumeRoleArn string) pollerInput {
	queries := make([]queryVarInput, len(cfg.Queries))
	for i, q := range cfg.Queries {
		qv := queryVarInput{
			Namespace:   q.Namespace,
			MetricNames: q.MetricNames,
		}
		if len(q.Dimensions) > 0 {
			qv.Dimensions = make([]dimensionVarInput, len(q.Dimensions))
			for j, d := range q.Dimensions {
				qv.Dimensions[j] = dimensionVarInput(d)
			}
		}
		if q.ResourceFilter != nil {
			rf := &resourceFilterInput{
				ResourceType:  q.ResourceFilter.ResourceType,
				Pattern:       q.ResourceFilter.Pattern,
				DimensionName: q.ResourceFilter.DimensionName,
			}
			if len(q.ResourceFilter.TagFilters) > 0 {
				rf.TagFilters = make([]tagFilterInput, len(q.ResourceFilter.TagFilters))
				for k, tf := range q.ResourceFilter.TagFilters {
					rf.TagFilters[k] = tagFilterInput(tf)
				}
			}
			qv.ResourceFilter = rf
		}
		queries[i] = qv
	}

	input := pollerInput{
		Name:         cfg.Name,
		DatastreamId: cfg.DatastreamId,
		Interval:     cfg.Interval,
		CloudWatchMetricsConfig: &cwMetricsConfigInput{
			Period:        fmt.Sprintf("%d", cfg.Period),
			Delay:         fmt.Sprintf("%d", cfg.Delay),
			Region:        region,
			AssumeRoleArn: assumeRoleArn,
			Queries:       queries,
		},
	}

	if cfg.Retries != nil {
		s := fmt.Sprintf("%d", *cfg.Retries)
		input.Retries = &s
	}

	return input
}

type gqlClient struct {
	httpClient        *http.Client
	observeAccountID  string
	observeDomainName string
	logger            logr.Logger
}

func (c *gqlClient) execute(token, query string, variables interface{}) ([]byte, error) {
	fullToken := fmt.Sprintf(bearerTokenFormat, c.observeAccountID, token)

	jsonData, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GQL request: %w", err)
	}

	host := fmt.Sprintf("%s.%s", c.observeAccountID, c.observeDomainName)
	url := fmt.Sprintf("https://%s/v1/meta", host)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fullToken)

	c.logger.V(4).Info("executing GQL request", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GQL request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GQL response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GQL request returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *gqlClient) createPoller(token, workspaceID string, cfg *PollerConfig, region, assumeRoleArn string) (string, error) {
	input := buildPollerInput(cfg, region, assumeRoleArn)
	variables := map[string]interface{}{
		"workspaceId": workspaceID,
		"poller":      input,
	}

	body, err := c.execute(token, createPollerMutation, variables)
	if err != nil {
		return "", err
	}

	var result createPollerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse createPoller response: %w", err)
	}
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("createPoller GQL error: %s", result.Errors[0].Message)
	}
	if result.Data.CreatePoller.ID == "" {
		return "", fmt.Errorf("createPoller returned empty ID")
	}

	c.logger.V(3).Info("created poller", "id", result.Data.CreatePoller.ID, "name", result.Data.CreatePoller.Name)
	return result.Data.CreatePoller.ID, nil
}

func (c *gqlClient) updatePoller(token, pollerID string, cfg *PollerConfig, region, assumeRoleArn string) error {
	input := buildPollerInput(cfg, region, assumeRoleArn)
	variables := map[string]interface{}{
		"id":     pollerID,
		"poller": input,
	}

	body, err := c.execute(token, updatePollerMutation, variables)
	if err != nil {
		return err
	}

	var result updatePollerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse updatePoller response: %w", err)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("updatePoller GQL error: %s", result.Errors[0].Message)
	}

	c.logger.V(3).Info("updated poller", "id", result.Data.UpdatePoller.ID)
	return nil
}

func (c *gqlClient) deletePoller(token, pollerID string) error {
	variables := map[string]interface{}{
		"id": pollerID,
	}

	body, err := c.execute(token, deletePollerMutation, variables)
	if err != nil {
		return err
	}

	var result deletePollerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse deletePoller response: %w", err)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("deletePoller GQL error: %s", result.Errors[0].Message)
	}

	c.logger.V(3).Info("deleted poller", "id", pollerID)
	return nil
}
