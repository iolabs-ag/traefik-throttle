package traefik_throttle

import (
	"net/http"
	"sync"
	"time"
)

// Throttle struct holds the global and endpoint-specific configurations,
// as well as the state for rate limiting.
type Throttle struct {
	globalConfig    *Config
	endpointConfigs map[string]map[string]*Config // Per-endpoint configurations by method
	next            http.Handler
	name            string

	limits     map[string]map[string]*endpointState // Supports per-method state
	userLimits map[string]map[string]*UserState     // Per-endpoint and method user limits
	limitsLock sync.RWMutex
}

// endpointState struct holds the state for each endpoint's rate limiting.
type endpointState struct {
	maxRequests int
	maxQueue    int

	retryCount    int
	retryDelay    time.Duration
	requestsCount int
	queueCount    int
	mutex         sync.RWMutex
}

// UserState struct holds the state for each user's rate limiting.
type UserState struct {
	maxRequests   int
	retryDelay    time.Duration
	requestsCount int
	mutex         sync.RWMutex
}
