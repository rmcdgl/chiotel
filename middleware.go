package chiotel

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

const version = "0.9.0"
const instrumentationName = "github.com/rmcdgl/chiotel"

// DefaultMiddleware is a ready to use middleware that uses the global tracer provider
func DefaultMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Wrap the ResponseWriter in order to view status without mutating it.
		// Use chi middleware rather than github.com/felixge/httpsnoop used in
		// other packages
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		tracer := resolveTracer(r.Context())
		carrier := propagation.HeaderCarrier(r.Header)
		prop := otel.GetTextMapPropagator()
		ctx := prop.Extract(r.Context(), carrier)

		// Create a temporary name based on what we know so far, append the route
		// to it later.
		name := "HTTP " + r.Method
		opts := []trace.SpanStartOption{
			trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
			trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
			trace.WithSpanKind(trace.SpanKindServer),
		}
		ctx, span := tracer.Start(ctx, name, opts...)
		defer span.End()
		r = r.WithContext(ctx)

		next.ServeHTTP(ww, r)

		// Chi builds the route path as it traverses the interal routing tree, so
		// the full path is only available once the request has been served.
		route := chi.RouteContext(r.Context()).RoutePattern()
		attrs := semconv.HTTPServerAttributesFromHTTPRequest("", route, r)
		attrs = append(attrs, semconv.HTTPAttributesFromHTTPStatusCode(ww.Status())...)
		span.SetAttributes(attrs...)

		// For when the request doesn't match any route. This is the same as in
		// github.com/open-telemetry/opentelemetry-go-contrib
		if route == "" {
			route = "route not found"
		}
		span.SetName(name + " " + route)

		code, desc := semconv.SpanStatusFromHTTPStatusCode(ww.Status())
		span.SetStatus(code, desc)

	})
}

// resolveTracer checks if there is an existing tracer in the given context,
// otherwise it creates a new one
func resolveTracer(ctx context.Context) trace.Tracer {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		return span.TracerProvider().Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(version),
			trace.WithSchemaURL(semconv.SchemaURL),
		)
	}
	return otel.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

}
