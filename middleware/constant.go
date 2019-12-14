package middleware

// Request ID constants
const (
	CtxRequestID    = "requestID"
	HeaderRequestID = "X-Request-ID"
)

// OpenTracing constants
const (
	CtxTraceSpan          = "traceSpan"
	CtxTraceOperationName = "traceOperationName"
	BaggageRequestID      = "requestID"
	TagHTTPURL            = "http.URL"
	TagHTTPMethod         = "http.method"
	TagHTTPStatusCode     = "http.statusCode"
	TagHTTPUserAgent      = "http.userAgent"
	TagHTTPClientIP       = "http.clientIP"

	unknownOperation = "unknown"
)
