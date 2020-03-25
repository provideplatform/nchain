package orchestration

// GoogleOrchestrationProvider is a network.orchestration.API implementing the Google API
type GoogleOrchestrationProvider struct {
	region string
}

// InitGoogleOrchestrationProvider initializes and returns the Google Cloud Platform infrastructure orchestration provider
func InitGoogleOrchestrationProvider(credentials map[string]interface{}) *GoogleOrchestrationProvider {
	return nil
}
