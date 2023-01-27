package confluent

// SecretAdmin defines the secret with the db admin access details.
type SecretAdmin struct {
	// Confluent API Key
	APIKey string `json:"cloud_api_key"`
	// Confluent API Secret
	APISecret string `json:"cloud_api_secret"`
}

// SecretUser defines the secret with db user access details.
// The map of attributes must include
// user 	<- API Key
// password <- API Secret
type SecretUser map[string]string
