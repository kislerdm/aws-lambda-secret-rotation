package neon

// SecretAdmin defines the secret with the db admin access details.
type SecretAdmin struct {
	Token string `json:"token"`
}

// SecretUser defines the secret with db user access details.
type SecretUser struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	ProjectID    string `json:"project_id"`
	BranchID     string `json:"branch_id"`
	DatabaseName string `json:"dbname"`
}
