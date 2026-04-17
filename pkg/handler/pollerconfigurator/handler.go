package pollerconfigurator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-logr/logr"
)

type Handler struct {
	Logger            logr.Logger
	ObserveAccountID  string
	ObserveDomainName string
	SecretName        string
	PollerConfigURI   string
	ExternalRoleName  string
	WorkspaceID       string
	Region            string
	AWSAccountID      string
}

func New(cfg *Config, logger logr.Logger) (Handler, error) {
	return Handler{
		Logger:            logger,
		ObserveAccountID:  cfg.ObserveAccountID,
		ObserveDomainName: cfg.ObserveDomainName,
		SecretName:        cfg.SecretName,
		PollerConfigURI:   cfg.PollerConfigURI,
		ExternalRoleName:  cfg.ExternalRoleName,
		WorkspaceID:       cfg.WorkspaceID,
		Region:            cfg.Region,
		AWSAccountID:      cfg.AWSAccountID,
	}, nil
}

func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	logger := h.Logger
	logger.V(4).Info("handling poller configurator request")

	req, err := h.parsePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}
	logger.V(3).Info("parsed request", "requestType", req.RequestType, "physicalResourceId", req.PhysicalResourceId)

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		if req.RequestType == "Delete" {
			logger.Error(err, "failed to load AWS config during delete, skipping poller cleanup")
			if reportErr := h.reportStatus(*req, true, "deleted (skipped poller cleanup: could not load AWS config)"); reportErr != nil {
				return nil, fmt.Errorf("failed to report success: %w", reportErr)
			}
			return []byte{}, nil
		}
		return nil, h.reportAndError("failed to load AWS config", req, err)
	}

	token, err := h.getSecretValue(ctx, awsCfg)
	if err != nil {
		if req.RequestType == "Delete" {
			logger.Error(err, "failed to retrieve GQL token during delete, skipping poller cleanup")
			if reportErr := h.reportStatus(*req, true, "deleted (skipped poller cleanup: could not retrieve token)"); reportErr != nil {
				return nil, fmt.Errorf("failed to report success: %w", reportErr)
			}
			return []byte{}, nil
		}
		return nil, h.reportAndError("failed to retrieve GQL token", req, err)
	}

	gql := &gqlClient{
		httpClient:        &http.Client{},
		observeAccountID:  h.ObserveAccountID,
		observeDomainName: h.ObserveDomainName,
		logger:            logger,
	}

	assumeRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", h.AWSAccountID, h.ExternalRoleName)

	switch req.RequestType {
	case "Create":
		return h.handleCreate(ctx, req, gql, token, assumeRoleArn)
	case "Update":
		return h.handleUpdate(ctx, req, gql, token, assumeRoleArn)
	case "Delete":
		return h.handleDelete(req, gql, token)
	default:
		return nil, h.reportAndError(fmt.Sprintf("unknown request type: %s", req.RequestType), req, nil)
	}
}

func (h Handler) handleCreate(ctx context.Context, req *Request, gql *gqlClient, token *string, assumeRoleArn string) ([]byte, error) {
	logger := h.Logger

	pollerCfg, err := h.downloadConfig(ctx)
	if err != nil {
		return nil, h.reportAndError("failed to download poller config", req, err)
	}
	logger.V(3).Info("downloaded poller config", "queries", len(pollerCfg.Queries))

	pollerCfg.Name = h.uniquePollerName(pollerCfg.Name)

	pollerID, err := gql.createPoller(*token, h.WorkspaceID, pollerCfg, h.Region, assumeRoleArn)
	if err != nil {
		return nil, h.reportAndError("failed to create poller", req, err)
	}

	logger.V(3).Info("poller created", "id", pollerID)
	if err := h.reportStatusWithPhysicalID(*req, true, "poller created successfully", pollerID); err != nil {
		return nil, fmt.Errorf("failed to report success: %w", err)
	}
	return []byte{}, nil
}

func (h Handler) handleUpdate(ctx context.Context, req *Request, gql *gqlClient, token *string, assumeRoleArn string) ([]byte, error) {
	logger := h.Logger

	pollerID := req.PhysicalResourceId
	if pollerID == "" {
		return nil, h.reportAndError("update requires PhysicalResourceId (poller ID)", req, nil)
	}

	pollerCfg, err := h.downloadConfig(ctx)
	if err != nil {
		return nil, h.reportAndError("failed to download poller config", req, err)
	}
	logger.V(3).Info("downloaded poller config for update", "queries", len(pollerCfg.Queries))

	pollerCfg.Name = h.uniquePollerName(pollerCfg.Name)

	if err := gql.updatePoller(*token, pollerID, pollerCfg, h.Region, assumeRoleArn); err != nil {
		return nil, h.reportAndError("failed to update poller", req, err)
	}

	logger.V(3).Info("poller updated", "id", pollerID)
	if err := h.reportStatusWithPhysicalID(*req, true, "poller updated successfully", pollerID); err != nil {
		return nil, fmt.Errorf("failed to report success: %w", err)
	}
	return []byte{}, nil
}

func (h Handler) handleDelete(req *Request, gql *gqlClient, token *string) ([]byte, error) {
	logger := h.Logger

	pollerID := req.PhysicalResourceId
	if pollerID == "" {
		logger.V(3).Info("no PhysicalResourceId on delete, reporting success")
		if err := h.reportStatus(*req, true, "no poller to delete"); err != nil {
			return nil, fmt.Errorf("failed to report success on empty delete: %w", err)
		}
		return []byte{}, nil
	}

	if err := gql.deletePoller(*token, pollerID); err != nil {
		logger.Error(err, "failed to delete poller, continuing", "id", pollerID)
	}

	logger.V(3).Info("poller deleted", "id", pollerID)
	if err := h.reportStatus(*req, true, "poller deleted successfully"); err != nil {
		return nil, fmt.Errorf("failed to report success: %w", err)
	}
	return []byte{}, nil
}

func (h Handler) uniquePollerName(baseName string) string {
	return fmt.Sprintf("%s-%s-%s", baseName, h.AWSAccountID, h.Region)
}

func (h Handler) downloadConfig(ctx context.Context) (*PollerConfig, error) {
	return downloadPollerConfig(ctx, h.PollerConfigURI)
}

func (h Handler) getSecretValue(ctx context.Context, cfg aws.Config) (*string, error) {
	svc := secretsmanager.NewFromConfig(cfg)
	secretValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &h.SecretName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret value: %w", err)
	}
	token := *secretValue.SecretString
	return &token, nil
}

func (h Handler) parsePayload(payload []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}
	return &req, nil
}

func (h Handler) reportAndError(reason string, request *Request, err error) error {
	cfnReason := reason
	if err != nil {
		cfnReason = fmt.Sprintf("%s: %v", reason, err)
	}
	reportErr := h.reportStatus(*request, false, cfnReason)
	if reportErr != nil {
		if err != nil {
			return fmt.Errorf("failed to report status: %w, while reporting error, %s: %w", reportErr, reason, err)
		}
		return fmt.Errorf("failed to report status: %w, reason: %s", reportErr, reason)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", reason, err)
	}
	return fmt.Errorf("%s", reason)
}

func (h Handler) reportStatus(request Request, success bool, reason string) error {
	return h.reportStatusWithPhysicalID(request, success, reason, request.PhysicalResourceId)
}

func (h Handler) reportStatusWithPhysicalID(request Request, success bool, reason, physicalResourceID string) error {
	logger := h.Logger

	statusString := "FAILED"
	if success {
		statusString = "SUCCESS"
	}

	if physicalResourceID == "" {
		physicalResourceID = "lambda-pollerconfigurator"
	}

	resp := CfResponse{
		Status:             statusString,
		PhysicalResourceId: physicalResourceID,
		Reason:             reason,
		StackId:            request.StackId,
		RequestId:          request.RequestId,
		LogicalResourceId:  request.LogicalResourceId,
	}

	body, _ := json.Marshal(resp)
	logger.V(4).Info("reporting status to cloudformation", "response", resp)

	req, _ := http.NewRequest("PUT", request.ResponseURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send response to cloudformation: %w", err)
	}

	return nil
}
