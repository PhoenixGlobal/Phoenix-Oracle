package presenters

type ServiceLogConfigResource struct {
	JAID
	ServiceName []string `json:"serviceName"`
	LogLevel    []string `json:"logLevel"`
}

func (r ServiceLogConfigResource) GetName() string {
	return "serviceLevelLogs"
}
