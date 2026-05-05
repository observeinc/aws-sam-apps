package pollerconfigurator

type Request struct {
	RequestId          string             `json:"RequestId"`
	StackId            string             `json:"StackId"`
	RequestType        string             `json:"RequestType"`
	LogicalResourceId  string             `json:"LogicalResourceId"`
	PhysicalResourceId string             `json:"PhysicalResourceId"`
	ResourceProperties ResourceProperties `json:"ResourceProperties"`
	ResourceType       string             `json:"ResourceType"`
	ResponseURL        string             `json:"ResponseURL"`
}

type ResourceProperties struct {
	ServiceToken string `json:"ServiceToken"`
	StackName    string `json:"StackName"`
}
