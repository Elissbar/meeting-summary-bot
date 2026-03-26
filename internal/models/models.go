package models

type SaluteAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type SaluteUploadResponse struct {
	Status int `json:"status"`
	Result result `json:"result"`
}

type result struct {
	RequestFileID string `json:"request_file_id"`
}