# Response formats

Generated `httpapi` packages use two plain conventions:

- **Success:** the resource itself as JSON, e.g. `{"id":1,"name":"Andryan"}` with `200` or `201`.
- **Errors:** [RFC 9457 problem details](https://www.rfc-editor.org/rfc/rfc9457) with
  `Content-Type: application/problem+json`:

  ```json
  {
    "type": "about:blank",
    "title": "Not Found",
    "status": 404,
    "detail": "order: not found",
    "instance": "/orders/99"
  }
  ```

Why this default: RFC 9457 is the IETF standard for HTTP API errors (it replaces RFC 7807). API
gateways, OpenAPI tooling and HTTP clients understand it out of the box, and the HTTP status code
stays the single source of truth for success or failure.

Everything lives in two functions in `httpapi/respond.go`, `writeJSON` and `writeProblem`. To use
a different format, change only those two.

## Alternative: a JSON envelope (Tokopedia style)

Many large Indonesian and Southeast Asian codebases (for example Tokopedia's) wrap every
response in one envelope with `header`, `status`, `data` and `errors`:

```json
{
  "header": { "total_data": 1, "process_time": 0.0042, "meta": {} },
  "status": { "error_code": 0, "message": "OK" },
  "data": { "id": 1, "name": "Andryan" }
}
```

```json
{
  "header": { "process_time": 0.0011 },
  "status": { "error_code": 404, "message": "Not Found" },
  "errors": [
    { "code": "404", "title": "Not Found", "detail": "order: not found", "object": { "text": null, "type": 0 } }
  ]
}
```

**When it fits:**
- You must match an existing API contract or client SDK that already expects the envelope.
- Clients need response metadata (`process_time`, pagination in `meta`) on every call.

**Trade-offs:** clients and tools can't rely on the status code or content type alone, every
consumer has to unwrap `data`, and there's no shared standard for the error shape.

### Switching a generated domain to the envelope

Replace `writeJSON` and `writeProblem` in `httpapi/respond.go` (and keep the handlers as they are):

```go
// Envelope is the standard response wrapper.
type Envelope struct {
	Header *Header  `json:"header,omitempty"`
	Status *Status  `json:"status,omitempty"`
	Data   any      `json:"data,omitempty"`
	Errors []Error  `json:"errors,omitempty"`
}

type Header struct {
	TotalData   int            `json:"total_data,omitempty"`
	ProcessTime float64        `json:"process_time"`
	Meta        map[string]any `json:"meta,omitempty"`
}

type Status struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

type Error struct {
	Code   string      `json:"code"`
	Title  string      `json:"title"`
	Detail string      `json:"detail"`
	Object ErrorObject `json:"object"`
}

type ErrorObject struct {
	Text []string `json:"text"`
	Type int64    `json:"type"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	writeEnvelope(w, status, Envelope{
		Header: &Header{TotalData: 1},
		Status: &Status{Message: http.StatusText(status)},
		Data:   v,
	})
}

func writeProblem(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, detail := http.StatusInternalServerError, ""
	switch {
	case errors.Is(err, order.ErrNotFound):
		status, detail = http.StatusNotFound, err.Error()
	case errors.Is(err, order.ErrInvalid):
		status, detail = http.StatusBadRequest, err.Error()
	default:
		logger.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "error", err)
	}
	writeEnvelope(w, status, Envelope{
		Status: &Status{ErrorCode: status, Message: http.StatusText(status)},
		Errors: []Error{{Code: strconv.Itoa(status), Title: http.StatusText(status), Detail: detail}},
	})
}

func writeEnvelope(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
```

To fill `process_time`, record `start := time.Now()` in a middleware, store it in the request
context, and read it in `writeEnvelope`. Then update `httpapi/handlers_test.go`, which currently
expects `application/problem+json` for errors.
