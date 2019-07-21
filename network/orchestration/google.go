package orchestration

// GoogleOrchestrationProvider is a network.OrchestrationAPI implementing the Google API
type GoogleOrchestrationProvider struct {
	region string
}

// InitGoogleOrchestrationProvider initializes and returns the Google Cloud Platform infrastructure orchestration provider
func InitGoogleOrchestrationProvider(credentials map[string]interface{}) *GoogleOrchestrationProvider {
	return nil
}
