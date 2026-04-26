package api

// ErrorResponse is the standard machine-readable API error envelope.
type ErrorResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// OKResponse is a generic success envelope.
type OKResponse struct {
	Success bool `json:"success"`
}
