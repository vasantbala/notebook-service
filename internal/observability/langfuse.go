// Package observability provides OpenTelemetry tracing that exports to Langfuse
// via its native OTLP endpoint. Langfuse maps OTLP spans onto its Trace/Generation
// model automatically when the correct resource attributes are set.
//
// Usage:
//
// tp, err := observability.Init(ctx, cfg.Langfuse)
// defer tp.Shutdown(ctx)
//
// tracer := observability.Tracer()
// ctx, span := tracer.Start(ctx, "notebook-chat")
// defer span.End()
package observability

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/vasantbala/notebook-service/internal/config"
)

const tracerName = "notebook-service"

// Init configures the global OpenTelemetry TracerProvider to export to Langfuse
// over OTLP/HTTP. Returns a no-op provider (and nil error) when Langfuse credentials
// are not configured so the service starts cleanly without observability.
func Init(ctx context.Context, cfg config.LangfuseConfig) (*sdktrace.TracerProvider, error) {
	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		// Observability disabled — install a no-op provider so traces compile fine.
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return nil, nil
	}

	// Langfuse OTLP endpoint: <host>/api/public/otel
	endpoint := cfg.Host + "/api/public/otel"

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		// Langfuse uses HTTP Basic auth: public_key:secret_key
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey+":"+cfg.SecretKey)),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("notebook-service"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

// Tracer returns the package-level tracer. Safe to call before Init (returns no-op).
func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// SpanSetUser tags a span with the authenticated user ID.
func SpanSetUser(span trace.Span, userID string) {
	span.SetAttributes(attribute.String("langfuse.user.id", userID))
}

// SpanSetInput tags a span with the user's input text.
func SpanSetInput(span trace.Span, input string) {
	span.SetAttributes(attribute.String("langfuse.input", truncate(input, 1000)))
}

// SpanSetOutput tags a span with the assistant's output text.
func SpanSetOutput(span trace.Span, output string) {
	span.SetAttributes(attribute.String("langfuse.output", truncate(output, 2000)))
}

// SpanSetModel tags a span with the model name (used by Langfuse's generation view).
func SpanSetModel(span trace.Span, model string) {
	span.SetAttributes(attribute.String("gen_ai.request.model", model))
}

// SpanSetTokens tags a span with prompt and completion token counts.
func SpanSetTokens(span trace.Span, promptTokens, completionTokens int) {
	span.SetAttributes(
		attribute.Int("gen_ai.usage.input_tokens", promptTokens),
		attribute.Int("gen_ai.usage.output_tokens", completionTokens),
	)
}

// SpanSetRAGChunks tags a span with the number of retrieved chunks.
func SpanSetRAGChunks(span trace.Span, count int) {
	span.SetAttributes(attribute.Int("rag.chunk_count", count))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
