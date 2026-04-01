package model

// EventResponse — POST /events success response
type EventResponse struct {
	Status    string `json:"status" example:"accepted"`
	EventHash string `json:"event_hash" example:"22f0e08a61d49a38798bec697afec95ffc00ba8af2e29639d4e91a15e1dbcea3"`
}

// BulkResponse — POST /events/bulk success response
type BulkResponse struct {
	Accepted   int `json:"accepted" example:"98"`
	Rejected   int `json:"rejected" example:"1"`
	Duplicates int `json:"duplicates" example:"1"`
}

// ErrorResponse — error response
type ErrorResponse struct {
	Error string `json:"error" example:"event_name is required"`
}

// HealthResponse — GET /health response
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message" example:"The API is up and running"`
}
