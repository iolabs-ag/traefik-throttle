# Traefik Middleware: Rate Limiter

The **Rate Limiter** middleware is designed to control traffic by limiting the number of requests and managing queues for each endpoint and user. This middleware is implemented in Go and provides flexible configurations to handle various traffic scenarios.

This middleware is based on another throttle middleware: [joegarb/traefik-throttle](https://github.com/joegarb/traefik-throttle)

## Features
- Rate limiting per endpoint and HTTP method.
- User-specific rate limits with queuing mechanisms.
- Dynamic retry mechanisms for queued requests.
- Centralized logging for debugging, warnings, and errors.
- Configurable via YAML files.

## Configuration
The Rate Limiter middleware requires a configuration file specifying global and endpoint-specific rate limiting rules.

### Example YAML Configuration
```yaml
maxRequests: 10
maxQueue: 5
retryCount: 3
retryDelay: "500ms"
userMaxRequests: 2
userRetryDelay: "1s"
endpoints:
  "/api/v1/resource":
    GET:
      maxRequests: 20
      maxQueue: 10
      retryCount: 5
      retryDelay: "200ms"
      userMaxRequests: 5
      userRetryDelay: "500ms"
    POST:
      maxRequests: 10
      maxQueue: 5
      retryCount: 3
      retryDelay: "300ms"
      userMaxRequests: 3
      userRetryDelay: "1s"
```

### Fields
- **maxRequests**: Maximum number of requests allowed globally.
- **maxQueue**: Maximum number of queued requests.
- **retryCount**: Number of retries before rejecting a request.
- **retryDelay**: Delay between retries.
- **userMaxRequests**: Maximum requests allowed per user.
- **userRetryDelay**: Delay for user-specific retries.

## Usage
### Installation as Traefik Plugin

To use the Rate Limiter middleware as a Traefik plugin, follow these steps:

1. **Enable the plugin** in your Traefik configuration:

   ```yaml
   experimental:
     plugins:
       rateLimiter:
         moduleName: "github.com/iolabs-ag/ratelimiter"
         version: "v1.0.0"
   ```

2. **Configure the middleware** in the Traefik dynamic configuration file:

   ```yaml
   http:
     middlewares:
       rate-limiter:
         plugin:
           rateLimiter:
             maxRequests: 10
             maxQueue: 5
             retryCount: 3
             retryDelay: "500ms"
             userMaxRequests: 2
             userRetryDelay: "1s"
             endpoints:
               "/api/v1/resource":
                 GET:
                   maxRequests: 20
                   maxQueue: 10
                   retryCount: 5
                   retryDelay: "200ms"
                   userMaxRequests: 5
                   userRetryDelay: "500ms"
   ```

3. **Attach the middleware** to a Traefik router:

   ```yaml
   http:
     routers:
       my-router:
         rule: "PathPrefix(`/api`)"
         service: my-service
         middlewares:
           - rate-limiter
   ```

## Logging
The middleware uses centralized logging with levels:
- **DEBUG**: For detailed information about the middleware's operations.
- **WARNING**: For non-critical issues (e.g., invalid configurations).
- **ERROR**: For critical errors (e.g., invalid parameters).

## Contributing
Contributions are welcome! Please submit a pull request or open an issue for any improvements or bug reports.

## License
This middleware is licensed under the MIT License. See the LICENSE file for details.

