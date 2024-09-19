package metricsrecorder

// N.B. This is the response that the custom resource returns to cloudformation
type CfResponse struct {
	Status             string                 `json:"Status"`
	Reason             string                 `json:"Reason,omitempty"`
	PhysicalResourceId string                 `json:"PhysicalResourceId,omitempty"`
	StackId            string                 `json:"StackId"`
	RequestId          string                 `json:"RequestId"`
	LogicalResourceId  string                 `json:"LogicalResourceId"`
	Data               map[string]interface{} `json:"Data,omitempty"`
}
