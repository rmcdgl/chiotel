package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmcdgl/chiotel"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

func TestDefaultMiddleware(t *testing.T) {
	tests := []struct {
		method    string
		target    string
		assertion func(*testing.T, string, string, sdktrace.ReadOnlySpan)
	}{
		{
			method:    http.MethodGet,
			target:    "/posts/123",
			assertion: assertSpan("/posts/{id}", codes.Unset, http.StatusOK),
		},
		{
			method:    http.MethodPut,
			target:    "/posts/123",
			assertion: assertSpan("/posts/{id}", codes.Unset, http.StatusAccepted),
		},
		{
			method:    http.MethodGet,
			target:    "/error",
			assertion: assertSpan("/error", codes.Error, http.StatusInternalServerError),
		},
	}

	for _, test := range tests {
		sr, r := newTestEnvironment(t)
		req := httptest.NewRequest(test.method, test.target, http.NoBody)
		r.ServeHTTP(httptest.NewRecorder(), req)
		require.Len(t, sr.Ended(), 1)
		test.assertion(t, test.method, test.target, sr.Ended()[0])
	}
}

func newTestServer() *chi.Mux {
	r := chi.NewRouter()
	r.Use(chiotel.DefaultMiddleware)
	r.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Put("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	return r
}

func newTestEnvironment(t *testing.T) (*tracetest.SpanRecorder, *chi.Mux) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })
	return sr, newTestServer()
}

func assertSpan(path string, otelCode codes.Code, httpCode int) func(*testing.T, string, string, sdktrace.ReadOnlySpan) {
	return func(t *testing.T, method, target string, span sdktrace.ReadOnlySpan) {
		name := "HTTP " + method
		if path != "" {
			name += " " + path
		}
		assert.Equal(t, name, span.Name())
		assert.Equal(t, trace.SpanKindServer, span.SpanKind())

		status := span.Status()
		assert.Equal(t, otelCode, status.Code)
		assert.Equal(t, "", status.Description)

		attrs := span.Attributes()
		assert.Contains(t, attrs, semconv.HTTPMethodKey.String(method))
		assert.Contains(t, attrs, semconv.HTTPTargetKey.String(target))
		assert.Contains(t, attrs, semconv.HTTPRouteKey.String(path))
		assert.Contains(t, attrs, semconv.HTTPSchemeHTTP)
		assert.Contains(t, attrs, semconv.HTTPStatusCodeKey.Int(httpCode))
		assert.Contains(t, attrs, semconv.HTTPFlavorHTTP11)
		assert.Contains(t, attrs, semconv.NetTransportTCP)

		keys := make(map[attribute.Key]struct{}, len(attrs))
		for _, a := range attrs {
			keys[a.Key] = struct{}{}
		}

		// These key values are potentially dynamic. Test an attribute
		// with this key is set regardless of its value.
		wantKeys := []attribute.Key{
			semconv.HTTPHostKey,
			semconv.NetPeerIPKey,
			semconv.NetPeerPortKey,
			semconv.NetHostNameKey,
		}
		for _, k := range wantKeys {
			assert.Contains(t, keys, k)
		}
	}
}
