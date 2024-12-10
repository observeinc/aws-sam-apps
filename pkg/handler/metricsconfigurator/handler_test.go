package metricsconfigurator

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
)

var service_list = []string{"AWS/EC2", "AWS/EBS", "AWS/S3"}
var testRequestType = "Create"
var testStackId = "arn:aws:cloudformation:us-east-1:123456789012:stack/MyStack/guid"
var testRequestId = "unique_id"
var testResourceType = "Custom::MyCustomResource"
var testLogicalResourceId = "MyCustomResource"

func TestConvertToMetricStreamFilters(t *testing.T) {
	tests := []struct {
		name        string
		metricsList []MetricsListItem
		expected    []types.MetricStreamFilter
	}{
		{
			name: "Single namespace with multiple metrics",
			metricsList: []MetricsListItem{
				{
					Namespace:   "AWS/EC2",
					MetricNames: []string{"CPUUtilization", "NetworkIn", "NetworkOut"},
				},
			},
			expected: []types.MetricStreamFilter{
				{
					Namespace:   &service_list[0],
					MetricNames: []string{"CPUUtilization", "NetworkIn", "NetworkOut"},
				},
			},
		},
		{
			name: "Multiple namespaces with different metrics",
			metricsList: []MetricsListItem{
				{
					Namespace:   service_list[0],
					MetricNames: []string{"CPUUtilization", "NetworkIn"},
				},
				{
					Namespace:   service_list[1],
					MetricNames: []string{"VolumeReadOps", "VolumeWriteOps"},
				},
			},
			expected: []types.MetricStreamFilter{
				{
					Namespace:   &service_list[0],
					MetricNames: []string{"CPUUtilization", "NetworkIn"},
				},
				{
					Namespace:   &service_list[1],
					MetricNames: []string{"VolumeReadOps", "VolumeWriteOps"},
				},
			},
		},
		{
			name:        "Empty metrics list",
			metricsList: []MetricsListItem{},
			expected:    []types.MetricStreamFilter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToMetricStreamFilters(tt.metricsList)

			if isSame := reflect.DeepEqual(result, tt.expected); !isSame {
				t.Errorf("convertToMetricStreamFilters() returned unexpected result, got: %+v\n, expected: %+v\n", result, tt.expected)
			}
		})
	}
}

func TestMakeMetricGroups(t *testing.T) {

	// namespace a: 500 metrics, namespace b: 600 metrics
	// namespace c: 100 metrics

	// the algorithm should put a in the first stream,
	// and b and c in the second

	metricsA := make([]string, 500)
	metricsB := make([]string, 600)
	metricsC := make([]string, 100)

	for i := range metricsA {
		metricsA[i] = fmt.Sprintf("metricA-%d", i)
	}

	for i := range metricsB {
		metricsB[i] = fmt.Sprintf("metricB-%d", i)
	}

	for i := range metricsC {
		metricsC[i] = fmt.Sprintf("metricC-%d", i)
	}

	namespaceA := types.MetricStreamFilter{
		Namespace:   &service_list[0],
		MetricNames: metricsA,
	}

	namespaceB := types.MetricStreamFilter{
		Namespace:   &service_list[1],
		MetricNames: metricsB,
	}

	namespaceC := types.MetricStreamFilter{
		Namespace:   &service_list[2],
		MetricNames: metricsC,
	}

	expectedStreams := [][]types.MetricStreamFilter{
		{
			namespaceA,
		},
		{
			namespaceB,
			namespaceC,
		},
	}

	// Create a mock Handler
	h := Handler{
		Logger: logr.Discard(), // Use a discard logger for testing
	}

	tests := []struct {
		name        string
		metricsList []types.MetricStreamFilter
		expected    [][]types.MetricStreamFilter
	}{
		{
			name:        "multiple namespaces with > 1000 metrics",
			metricsList: []types.MetricStreamFilter{namespaceA, namespaceB, namespaceC},
			expected:    expectedStreams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.makeMetricGroups(tt.metricsList)

			if isSame := reflect.DeepEqual(result, tt.expected); !isSame {
				t.Errorf("makeMetricGroups() returned unexpected result, got: %+v\n, expected: %+v\n", result, tt.expected)
			}
		})
	}
}

func TestParsePayload(t *testing.T) {

	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Create a mock Handler
	h := Handler{
		Logger: logr.Discard(), // Use a discard logger for testing
	}

	// Test cases
	testCases := []struct {
		name          string
		payload       []byte
		expectedError bool
	}{
		{
			name: "Valid payload",
			payload: []byte(fmt.Sprintf(`{
				"RequestType": "%s",
				"ResponseURL": "%s",
				"StackId": "%s",
				"RequestId": "%s",
				"ResourceType": "%s",
				"LogicalResourceId": "%s"
			}`, testRequestType, mockServer.URL, testStackId, testRequestId, testResourceType, testLogicalResourceId)),
			expectedError: false,
		},
		{
			name:          "Invalid JSON payload",
			payload:       []byte(`{"invalid": json`),
			expectedError: true,
		},
		{
			name:          "Empty payload",
			payload:       []byte{},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := h.parsePayload(tc.payload)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil for testcase %s", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v for testcase %s", err, tc.name)
				}

				// Additional checks for the valid case
				if req != nil {
					if req.RequestType != testRequestType {
						t.Errorf("Expected RequestType to be %s, got '%s'", testRequestType, req.RequestType)
					}
					if req.ResponseURL != mockServer.URL {
						t.Errorf("Expected ResponseURL to be '%s', got '%s'", mockServer.URL, req.ResponseURL)
					}
					if req.StackId != testStackId {
						t.Errorf("Expected StackId to be '%s', got '%s'", testStackId, req.StackId)
					}
					if req.RequestId != testRequestId {
						t.Errorf("Expected RequestId to be '%s', got '%s'", testRequestId, req.RequestId)
					}
					if req.ResourceType != testResourceType {
						t.Errorf("Expected ResourceType to be %s, got '%s'", testResourceType, req.ResourceType)
					}
					if req.LogicalResourceId != testLogicalResourceId {
						t.Errorf("Expected LogicalResourceId to be '%s', got '%s'", testLogicalResourceId, req.LogicalResourceId)
					}
				}
			}
		})
	}
}

// test parseResponse
func TestParseResponse(t *testing.T) {

	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	h := Handler{
		Logger: logr.Discard(), // Use a discard logger for testing
	}

	// Test cases
	testCases := []struct {
		name           string
		responseBody   []byte
		expectedResult []types.MetricStreamFilter
		expectedError  bool
	}{
		{
			name: "Valid response",
			responseBody: []byte(`{
				"data": {
					"datasource": {
						"name": "TestDatasource",
						"config": {
							"awsCollectionStackConfig": {
								"awsServiceMetricsList": [
									{
										"namespace": "AWS/EC2",
										"metricNames": ["CPUUtilization", "NetworkIn"]
									},
									{
										"namespace": "AWS/EBS",
										"metricNames": ["VolumeReadOps", "VolumeWriteOps"]
									}
								]
							}
						}
					}
				}
			}`),
			expectedResult: []types.MetricStreamFilter{
				{
					Namespace:   &service_list[0],
					MetricNames: []string{"CPUUtilization", "NetworkIn"},
				},
				{
					Namespace:   &service_list[1],
					MetricNames: []string{"VolumeReadOps", "VolumeWriteOps"},
				},
			},
			expectedError: false,
		},
		{
			name:           "Invalid JSON response",
			responseBody:   []byte(`{"invalid": json`),
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name:           "Empty response",
			responseBody:   []byte{},
			expectedResult: nil,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := h.parseResponse(tc.responseBody)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil for testcase %s", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v for testcase %s", err, tc.name)
				}

				// Check the result
				if !reflect.DeepEqual(result, tc.expectedResult) {
					t.Errorf("Unexpected result for testcase %s. Got: %+v, Expected: %+v", tc.name, result, tc.expectedResult)
				}
			}
		})
	}
}

// test getDatasource
func TestGetDatasource(t *testing.T) {
	// Create a handler with mocked values
	h := Handler{
		Logger:       logr.Discard(),
		AccountID:    "123456789",
		DatasourceID: "testDatasourceID",
	}

	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Prepare test data
	token := "testToken"
	observeDomainName := "observe-eng.com"

	// Create a mock client
	mockRoundTripper := &mockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Check the request
			if diff := cmp.Diff("POST", req.Method); diff != "" {
				t.Fatalf("incorrect method: %s", diff)
			}
			if diff := cmp.Diff("application/json", req.Header.Get("Content-Type")); diff != "" {
				t.Fatalf("incorrect content-type: %s", diff)
			}
			if diff := cmp.Diff("https://123456789.observe-eng.com/v1/meta", req.URL.String()); diff != "" {
				t.Fatalf("incorrect url: %s", diff)
			}
			if diff := cmp.Diff("Bearer 123456789 testToken", req.Header.Get("Authorization")); diff != "" {
				t.Fatalf("incorrect authorization: %s", diff)
			}

			// Prepare mock response
			mockResponse := &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
                    "data": {
                        "datasource": {
                            "name": "TestDatasource",
							"config": {
								"awsCollectionStackConfig": {
									"awsServiceMetricsList": [
										{
											"namespace": "AWS/EC2",
											"metricNames": ["CPUUtilization", "NetworkIn"]
										}
									]
								}
							}
                        }
                    }
                }`)),
			}
			return mockResponse, nil
		},
	}

	// Create a mock client with the mock round tripper
	mockClient := &http.Client{
		Transport: mockRoundTripper,
	}

	// Call getDatasource with the mock client
	result, err := h.getDatasource(&token, observeDomainName, mockClient)
	if err != nil {
		t.Fatalf("Error calling getDatasource: %v", err)
	}

	// Call parseResponse with the result
	filters, err := h.parseResponse(result)
	if err != nil {
		t.Fatalf("Error calling parseResponse: %v", err)
	}

	// expected
	expectedResult := []types.MetricStreamFilter{
		{
			Namespace:   &service_list[0],
			MetricNames: []string{"CPUUtilization", "NetworkIn"},
		},
	}

	// Check the result
	if !reflect.DeepEqual(filters, expectedResult) {
		t.Errorf("Unexpected result, got: %+v, Expected: %+v", result, expectedResult)
	}
}

type mockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}
