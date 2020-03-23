package common

// Configurable interface
type Configurable interface {
	DecryptedConfig() (map[string]interface{}, error)
	SetConfig(map[string]interface{})
	SetEncryptedConfig(map[string]interface{})
	SanitizeConfig()
	ParseConfig() map[string]interface{}
}
