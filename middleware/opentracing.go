package middleware

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

var anonymousFunctionMatcher = regexp.MustCompile(`\.func\d$`)

// OpenTracingMiddleware provide basic span extract ability
// and set request ID into span baggage and put span into gin.Context, auto finish at the tail of handler chain.
// WARNING: Won't execute trace work if use this middleware without or before register OpenTracing tracer
func OpenTracingMiddleware(handlerPackageName string) gin.HandlerFunc {
	if !opentracing.IsGlobalTracerRegistered() {
		fmt.Println("OpenTracing global tracer ever register yet")
		return nil
	}
	return func(c *gin.Context) {
		var span opentracing.Span
		// Extract trace context from HTTP Header
		traceContext, err := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		var opts []opentracing.StartSpanOption
		if err == nil {
			opts = append(opts, opentracing.ChildOf(traceContext))
		}
		span = opentracing.StartSpan(unknownOperation, opts...)
		span.SetBaggageItem(BaggageRequestID, c.GetString(CtxRequestID))

		c.Set(CtxTraceSpan, span)
		// Keep going on next handlers
		c.Next()

		// Getting operation name from context if manually set
		operationName := c.GetString(CtxTraceOperationName)
		if operationName == "" {
			operationName = getHandlerName(handlerPackageName, c.HandlerNames())
		}
		span.SetOperationName(operationName)
		c.Set(CtxTraceOperationName, operationName)
		// Set HTTP related information
		span.SetTag(TagHTTPURL, c.Request.URL.String())
		span.SetTag(TagHTTPMethod, c.Request.Method)
		span.SetTag(TagHTTPUserAgent, c.Request.UserAgent())
		span.SetTag(TagHTTPClientIP, c.ClientIP())
		span.SetTag(TagHTTPStatusCode, c.Writer.Status())
		// Report to server
		span.Finish()
	}
}

// getHandlerName returns the LAST non-anonymous function in given package as the handler name
// WARNING: This may cause performance issue while having a long handler calling chain.
func getHandlerName(handlerPackageName string, handlerNames []string) string {
	handlerName := unknownOperation
	keyword := handlerPackageName + "."
	// Reverse loop to speed up the progress
	for i := len(handlerNames) - 1; i >= 0; i-- {
		// bypass anonymous function(name) with the suffix .func<digit>
		if strings.Contains(handlerNames[i], keyword) &&
			!anonymousFunctionMatcher.MatchString(handlerNames[i]) {
			segs := strings.Split(handlerNames[i], "/")
			if segLength := len(segs); segLength > 0 {
				handlerName = segs[segLength-1]
			}
			break
		}
	}
	return handlerName
}

// GetOpenTracingSpanFromContext returns tracing span from context otherwise start a new span
func GetOpenTracingSpanFromContext(c *gin.Context) opentracing.Span {
	rawSpan, ok := c.Get(CtxTraceSpan)
	if ok {
		return rawSpan.(opentracing.Span)
	}
	return opentracing.StartSpan(unknownOperation)
}
