package constants

type ContextKey string

const (
	TraceIDKey      ContextKey = "trace_id"
	SpanIDKey       ContextKey = "span_id"
	Session         ContextKey = "session"
	ContentType                = "Content-Type"
	ContentTypeJSON            = "application/json"
	ContentJson                = "application/json"
)