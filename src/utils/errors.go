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

// InternalError represents a server-side failure (filesystem, D-Bus, network, /proc, etc.).
type InternalError struct {
	Message string
}

func (e *InternalError) Error() string { return e.Message }
