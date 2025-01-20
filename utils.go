package traefik_throttle

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"time"
)

// Centralized logging levels
const (
	LogLevelDebug   = "DEBUG"
	LogLevelWarning = "WARNING"
	LogLevelError   = "ERROR"
)

// log logs messages with a specific log level and context.
func log(level, message string, err error) {
	if err != nil {
		fmt.Printf("[%s]: %s: %v\n", level, message, err)
	} else {
		fmt.Printf("[%s]: %s\n", level, message)
	}
}

// parseDurationOrDefault parses a duration string or returns a default value on failure.
func parseDurationOrDefault(value string, defaultDuration time.Duration) (time.Duration, error) {
	if value == "" {
		return defaultDuration, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultDuration, err
	}
	return d, nil
}

// parseDurationWithFallback parses a duration string with a fallback value for errors.
func parseDurationWithFallback(value string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

// getUserIDFromJWT extracts the user ID from the JWT token in the request.
func getUserIDFromJWT(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", nil
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", nil
	}

	return userID, nil
}
