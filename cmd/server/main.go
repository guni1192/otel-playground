package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	// "go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("github.com/guni1192/otel-playground")
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func newExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("jaeger:4317"),
	)
}

func guniDevHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://guni.dev", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Warn("failed to create request: %v", err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Warn("failed to do request: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		http.Error(w, "guni.dev response status is not expected", http.StatusInternalServerError)
		logger.Warn("guni.dev respons status is not expected: %v", res.Status)
	}
	defer res.Body.Close()

	w.Write([]byte("guni.dev is OK!"))
	logger.Info(
		"http request",
		"status", res.StatusCode,
		"url", res.Request.URL.String(),
		"method", res.Request.Method,
	)
}

func main() {
	ctx := context.Background()
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String("otel-playground"),
		attribute.String("environment", "development"),
	))
	if err != nil {
		logger.Error("failed to create resource: %v", err)
		os.Exit(1)
	}

	exporter, err := newExporter(ctx)
	if err != nil {
		logger.Error("failed to create jaeger exporter: %v", err)
		os.Exit(1)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	mux := http.NewServeMux()
	mux.HandleFunc("/guni", guniDevHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})

	http.ListenAndServe(":8080", otelhttp.NewHandler(mux, "otel-playground", otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents)))
}
