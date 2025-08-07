# Golang SDK Architecture Design

## 1. Client Structure
- A main `Client` struct that holds the API key, base URL, and HTTP client.
- Resource-specific services (e.g., `TasksService`, `IndexesService`, `SearchService`, `EmbedService`, `ManageVideosService`) as fields within the `Client` struct. This mirrors the Node.js and Python SDKs' approach of exposing resources as properties.

## 2. Authentication
- The API key will be passed during the `Client` initialization.
- It will be included as an `X-API-KEY` header in all requests.

## 3. HTTP Client
- Use Go's standard `net/http` package for making HTTP requests.
- Implement a custom `http.Client` with configurable timeouts.
- A `do` method within the `Client` struct to handle common request logic (setting headers, making requests, handling responses).

## 4. Request and Response Models
- Define Go structs for all request and response bodies, mirroring the API's JSON structure.
- Use `json` tags for proper serialization/deserialization.

## 5. Error Handling
- Define custom error types for API-specific errors (e.g., `BadRequestError`, `UnauthorizedError`, `NotFoundError`, `TooManyRequestsError`, `InternalServerError`).
- Return `error` types from service methods, allowing callers to check for specific error types.

## 6. Asynchronous Operations
- Go's concurrency model (goroutines and channels) can be used for asynchronous operations if needed, but for a typical SDK, synchronous blocking calls are often sufficient and simpler to manage. If long-running tasks (like video uploads) require polling, a separate `WaitForDone` method can be implemented.

## 7. File Uploads
- For video uploads, use `multipart/form-data` encoding with `io.Reader` for efficient streaming of file content.

## 8. Best Practices
- **Security**: Ensure API key is handled securely (e.g., not logged, passed via environment variables).
- **Performance**: Use `http.Client` with connection pooling. Efficient JSON serialization/deserialization. Stream file uploads.
- **Readability and Maintainability**: Clear package structure, well-named functions and variables, comprehensive comments.
- **Testability**: Design components to be easily testable with mock HTTP clients.
- **Modularity**: Separate concerns into different packages (e.g., `client`, `tasks`, `indexes`, `models`, `errors`).

## 9. Project Structure
```
twelvelabs-go-sdk/
├── go.mod
├── go.sum
├── client.go
├── tasks.go
├── indexes.go
├── search.go
├── embed.go
├── managevideos.go
├── models.go
├── errors.go
└── README.md
```


