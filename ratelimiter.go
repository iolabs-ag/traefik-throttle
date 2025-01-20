package traefik_throttle

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const typeName = "Throttle"

// New creates a new Throttle instance with the provided configuration.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		config = CreateConfig()
	}

	if config.EndpointsConfigLocation != "" {
		if err := loadConfigFromFile(config); err != nil {
			log(LogLevelWarning, "failed to load endpoints config from file", err)
		}
	}

	retryDelay, err := parseDurationOrDefault(config.RetryDelay, time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("invalid global retry delay: %v", err)
	}
	config.retryDelayDuration = retryDelay

	userRetryDelay, err := parseDurationOrDefault(config.UserRetryDelay, time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid user retry delay: %v", err)
	}
	config.userRetryDelayDuration = userRetryDelay

	// Initialize endpoint-specific configurations
	limits := make(map[string]map[string]*endpointState)
	for endpoint, methodConfigs := range config.Endpoints {
		if limits[endpoint] == nil {
			limits[endpoint] = make(map[string]*endpointState)
		}
		for method, endpointConfig := range methodConfigs {
			retryDelay, err := parseDurationOrDefault(endpointConfig.RetryDelay, time.Millisecond)
			if err != nil {
				log(LogLevelWarning, "invalid retry delay for endpoint", err)
				retryDelay = time.Millisecond
			}
			endpointConfig.retryDelayDuration = retryDelay

			userRetryDelay, err := parseDurationOrDefault(endpointConfig.UserRetryDelay, time.Second)
			if err != nil {
				log(LogLevelWarning, "invalid user retry delay for endpoint", err)
				userRetryDelay = time.Second
			}
			endpointConfig.userRetryDelayDuration = userRetryDelay

			limits[endpoint][method] = &endpointState{
				maxRequests: endpointConfig.MaxRequests,
				maxQueue:    endpointConfig.MaxQueue,
				retryCount:  endpointConfig.RetryCount,
				retryDelay:  retryDelay,
			}
		}
	}

	return &Throttle{
		globalConfig:    config,
		endpointConfigs: config.Endpoints,
		next:            next,
		name:            name,
		limits:          limits,
		userLimits:      make(map[string]map[string]*UserState),
	}, nil
}

// ServeHTTP handles the incoming HTTP requests and applies rate limiting.
func (t *Throttle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	userID, _ := getUserIDFromJWT(req)

	if userID == "" {
		rw.Header().Add("x-throttle-level", "endpoint")
		endpointState := t.getEndpointConfig(req.URL.Path, req.Method)
		t.applyRateLimiting(rw, req, endpointState)
		return
	}

	rw.Header().Add("x-throttle-level", "user")
	if !t.applyUserLimits(rw, req.URL.Path, req.Method, userID) {
		return
	}

	endpointState := t.getEndpointConfig(req.URL.Path, req.Method)
	rw.Header().Add("x-throttle-level", "endpoint or global")
	t.applyRateLimiting(rw, req, endpointState)
}

// applyUserLimits applies rate limiting for a specific user, including a queuing mechanism.
func (t *Throttle) applyUserLimits(rw http.ResponseWriter, endpoint, method, userID string) bool {
	key := fmt.Sprintf("%s#%s", endpoint, method)
	t.limitsLock.RLock()
	if _, exists := t.userLimits[key]; !exists {
		t.limitsLock.RUnlock()
		t.limitsLock.Lock()
		if _, exists := t.userLimits[key]; !exists {
			t.userLimits[key] = make(map[string]*UserState)
		}
		t.limitsLock.Unlock()
		t.limitsLock.RLock()
	}

	userState, exists := t.userLimits[key][userID]
	t.limitsLock.RUnlock()
	if !exists {
		methodConfigs, methodExists := t.endpointConfigs[endpoint]
		var endpointConfig *Config
		if methodExists {
			endpointConfig = methodConfigs[method]
		}
		maxRequests := t.globalConfig.UserMaxRequests
		retryDelay := t.globalConfig.userRetryDelayDuration
		if endpointConfig != nil {
			if endpointConfig.UserMaxRequests > 0 {
				maxRequests = endpointConfig.UserMaxRequests
			}
			retryDelay = endpointConfig.userRetryDelayDuration
		}
		userState = &UserState{
			maxRequests:   maxRequests,
			retryDelay:    retryDelay,
			requestsCount: 0,
		}
		t.limitsLock.Lock()
		t.userLimits[key][userID] = userState
		t.limitsLock.Unlock()
	}

	// Manage user state with minimal lock time
	userState.mutex.Lock()
	defer userState.mutex.Unlock()

	if userState.requestsCount >= userState.maxRequests {
		log(LogLevelDebug, fmt.Sprintf("User %s exceeded max requests", userID), nil)
		rw.WriteHeader(http.StatusTooManyRequests)
		return false
	}

	userState.requestsCount++
	time.AfterFunc(userState.retryDelay, func() {
		t.limitsLock.Lock()
		defer t.limitsLock.Unlock()

		userState.mutex.Lock()
		userState.requestsCount--
		userState.mutex.Unlock()
	})

	return true
}

// applyRateLimiting applies rate limiting to requests for a specific endpoint and method.
func (t *Throttle) applyRateLimiting(rw http.ResponseWriter, req *http.Request, state *endpointState) {
	attempt := state.retryCount
	queued := false
	incrementedQueue := false

	for attempt >= 0 {
		state.mutex.Lock()
		if state.requestsCount < state.maxRequests {
			state.requestsCount++
			if queued {
				state.queueCount--
			}
			state.mutex.Unlock()

			defer func() {
				state.mutex.Lock()
				state.requestsCount--
				state.mutex.Unlock()
			}()

			t.next.ServeHTTP(rw, req)
			return
		}

		if !incrementedQueue {
			if state.queueCount < state.maxQueue {
				state.queueCount++
				incrementedQueue = true
				queued = true
			}
		} else {
			state.mutex.Unlock()
			time.Sleep(state.retryDelay)
			attempt--
			continue
		}

		if state.queueCount >= state.maxQueue {
			log(LogLevelDebug, "Queue limit reached for endpoint", nil)
			state.mutex.Unlock()
			rw.WriteHeader(http.StatusTooManyRequests)
			return
		}
		state.mutex.Unlock()

		time.Sleep(state.retryDelay)
		attempt--
	}

	if queued {
		state.mutex.Lock()
		state.queueCount--
		state.mutex.Unlock()
	}

	log(LogLevelDebug, "Request denied after all retry attempts", nil)
	rw.WriteHeader(http.StatusTooManyRequests)
}

// getEndpointConfig retrieves the rate limiting configuration for a specific endpoint and method.
func (t *Throttle) getEndpointConfig(path, method string) *endpointState {
	t.limitsLock.RLock()
	methodStates, exists := t.limits[path]
	t.limitsLock.RUnlock()
	if !exists {
		t.limitsLock.Lock()
		if _, exists := t.limits[path]; !exists {
			methodStates = make(map[string]*endpointState)
			t.limits[path] = methodStates
		}
		t.limitsLock.Unlock()
	}

	t.limitsLock.RLock()
	state, exists := methodStates[method]
	t.limitsLock.RUnlock()
	if !exists {
		state = &endpointState{
			maxRequests: t.globalConfig.MaxRequests,
			maxQueue:    t.globalConfig.MaxQueue,
			retryCount:  t.globalConfig.RetryCount,
			retryDelay:  t.globalConfig.retryDelayDuration,
		}
		t.limitsLock.Lock()
		methodStates[method] = state
		t.limitsLock.Unlock()
	}

	return state
}
