package neon

// SecretAdmin defines the secret with the db admin access details.
type SecretAdmin struct {
	// Confluent API Key
	APIKey string `json:"cloud_api_key"`
	// Confluent API Secret
	APISecret string `json:"cloud_api_secret"`
}

// SecretUser defines the secret with db user access details.
type SecretUser struct {
	ServiceAccountID string `json:"user_id"`
	ClusterID        string `json:"cluster_id"`
	// Kafka bootstrap server
	BootstrapServer string `json:"bootstrap_server"`
	// Kafka SASL user
	User string `json:"user"`
	// Kafka SASL password
	Password string `json:"password"`
}
