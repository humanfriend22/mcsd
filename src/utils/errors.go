package utils

// ValidationError represents a client-caused error (bad input, budget exceeded, port conflict, etc.).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// NotFoundError represents a resource that doesn't exist.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// ServerError represents a server-side failure (filesystem, D-Bus, network, /proc, etc.).
type ServerError struct {
	Message string
}

func (e *ServerError) Error() string { return e.Message }
