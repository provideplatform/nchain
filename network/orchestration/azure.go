package orchestration

// AzureOrchestrationProvider is a network.OrchestrationAPI implementing the Azure API
type AzureOrchestrationProvider struct {
	region string
}

// InitAzureOrchestrationProvider initializes and returns the Microsoft Azure infrastructure orchestration provider
func InitAzureOrchestrationProvider(credentials map[string]interface{}, region string) *AzureOrchestrationProvider {
	return nil
}
