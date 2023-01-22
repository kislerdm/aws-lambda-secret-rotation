package neon

// SecretAdmin defines the secret with the db admin access details.
type SecretAdmin struct {
	// Token Neon API token
	Token string `json:"token"`
}

// SecretUser defines the secret with db user access details.
type SecretUser struct {
	// User Neon role
	User string `json:"user"`
	// Password Neon role's access password
	Password string `json:"password"`
	// Host Neon endpoint URI to access database
	Host string `json:"host"`
	// ProjectID Neon project ID
	ProjectID string `json:"project_id"`
	// BranchID Neon branch ID
	BranchID string `json:"branch_id"`
	// DatabaseName Neon database name
	DatabaseName string `json:"dbname"`
}
