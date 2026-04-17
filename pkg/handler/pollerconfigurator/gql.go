package pollerconfigurator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
)

const bearerTokenFormat = "Bearer %s %s"

type graphQLRequest struct {
	Query string `json:"query"`
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

func buildQueriesGQL(queries []QueryConfig) string {
	var parts []string
	for _, q := range queries {
		var fields []string
		fields = append(fields, fmt.Sprintf(`namespace: %q`, q.Namespace))

		if len(q.MetricNames) > 0 {
			names := make([]string, len(q.MetricNames))
			for i, n := range q.MetricNames {
				names[i] = fmt.Sprintf("%q", n)
			}
			fields = append(fields, fmt.Sprintf(`metricNames: [%s]`, strings.Join(names, ", ")))
		}

		if len(q.Dimensions) > 0 {
			var dims []string
			for _, d := range q.Dimensions {
				if d.Value != "" {
					dims = append(dims, fmt.Sprintf(`{name: %q, value: %q}`, d.Name, d.Value))
				} else {
					dims = append(dims, fmt.Sprintf(`{name: %q}`, d.Name))
				}
			}
			fields = append(fields, fmt.Sprintf(`dimensions: [%s]`, strings.Join(dims, ", ")))
		}

		if q.ResourceFilter != nil {
			rf := q.ResourceFilter
			var rfFields []string
			if rf.ResourceType != "" {
				rfFields = append(rfFields, fmt.Sprintf(`resourceType: %q`, rf.ResourceType))
			}
			if rf.Pattern != "" {
				rfFields = append(rfFields, fmt.Sprintf(`pattern: %q`, rf.Pattern))
			}
			if rf.DimensionName != "" {
				rfFields = append(rfFields, fmt.Sprintf(`dimensionName: %q`, rf.DimensionName))
			}
			var tagParts []string
			for _, tf := range rf.TagFilters {
				if len(tf.Values) > 0 {
					vals := make([]string, len(tf.Values))
					for i, v := range tf.Values {
						vals[i] = fmt.Sprintf("%q", v)
					}
					tagParts = append(tagParts, fmt.Sprintf(`{key: %q, values: [%s]}`, tf.Key, strings.Join(vals, ", ")))
				} else {
					tagParts = append(tagParts, fmt.Sprintf(`{key: %q}`, tf.Key))
				}
			}
			rfFields = append(rfFields, fmt.Sprintf(`tagFilters: [%s]`, strings.Join(tagParts, ", ")))
			fields = append(fields, fmt.Sprintf(`resourceFilter: {%s}`, strings.Join(rfFields, ", ")))
		}

		parts = append(parts, fmt.Sprintf(`{%s}`, strings.Join(fields, ", ")))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func buildPollerInputGQL(cfg *PollerConfig, region, assumeRoleArn string) string {
	var fields []string
	if cfg.Name != "" {
		fields = append(fields, fmt.Sprintf(`name: %q`, cfg.Name))
	}
	if cfg.DatastreamId != "" {
		fields = append(fields, fmt.Sprintf(`datastreamId: %q`, cfg.DatastreamId))
	}
	if cfg.Interval != "" {
		fields = append(fields, fmt.Sprintf(`interval: %q`, cfg.Interval))
	}
	if cfg.Retries != nil {
		fields = append(fields, fmt.Sprintf(`retries: "%d"`, *cfg.Retries))
	}

	cwFields := []string{
		fmt.Sprintf(`period: "%d"`, cfg.Period),
		fmt.Sprintf(`delay: "%d"`, cfg.Delay),
		fmt.Sprintf(`region: %q`, region),
		fmt.Sprintf(`assumeRoleArn: %q`, assumeRoleArn),
		fmt.Sprintf(`queries: %s`, buildQueriesGQL(cfg.Queries)),
	}

	fields = append(fields, fmt.Sprintf(`cloudWatchMetricsConfig: {%s}`, strings.Join(cwFields, ", ")))
	return fmt.Sprintf(`{%s}`, strings.Join(fields, ", "))
}

func buildCreatePollerMutation(workspaceID string, cfg *PollerConfig, region, assumeRoleArn string) string {
	input := buildPollerInputGQL(cfg, region, assumeRoleArn)
	return fmt.Sprintf(`mutation { createPoller(workspaceId: %q, poller: %s) { id name } }`, workspaceID, input)
}

func buildUpdatePollerMutation(pollerID string, cfg *PollerConfig, region, assumeRoleArn string) string {
	input := buildPollerInputGQL(cfg, region, assumeRoleArn)
	return fmt.Sprintf(`mutation { updatePoller(id: %q, poller: %s) { id name } }`, pollerID, input)
}

func buildDeletePollerMutation(pollerID string) string {
	return fmt.Sprintf(`mutation { deletePoller(id: %q) { success } }`, pollerID)
}

type gqlClient struct {
	httpClient        *http.Client
	observeAccountID  string
	observeDomainName string
	logger            logr.Logger
}

func (c *gqlClient) execute(token, query string) ([]byte, error) {
	fullToken := fmt.Sprintf(bearerTokenFormat, c.observeAccountID, token)

	jsonData, err := json.Marshal(graphQLRequest{Query: query})
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
	query := buildCreatePollerMutation(workspaceID, cfg, region, assumeRoleArn)
	body, err := c.execute(token, query)
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
	query := buildUpdatePollerMutation(pollerID, cfg, region, assumeRoleArn)
	body, err := c.execute(token, query)
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
	query := buildDeletePollerMutation(pollerID)
	body, err := c.execute(token, query)
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
