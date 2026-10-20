// Package tracing wires an OpenTelemetry OTLP/gRPC trace exporter and a
// batched tracer provider. The exporter honors OTEL_EXPORTER_OTLP_ENDPOINT
// (and the full set of OTEL_* env vars the SDK understands) with no hard
// requirement that a collector be reachable at startup — if the endpoint is
// unset or the exporter fails to connect, traces are silently dropped and
// the API keeps serving. Health probes should not fail because of this.
package tracing

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const serviceName = "mechatracker-api"

// Init configures the global tracer provider and propagator. Returns a
// shutdown func that flushes pending spans; call it on server shutdown.
// If initialization fails or no endpoint is configured, Init logs a warning
// and returns a no-op shutdown; the caller should not treat this as fatal.
func Init(ctx context.Context, env, version string, log *zap.Logger) func(context.Context) error {
	noop := func(context.Context) error { return nil }

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		log.Info("otel: no OTLP endpoint configured; traces disabled")
		return noop
	}

	ctxDial, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exp, err := otlptracegrpc.New(ctxDial)
	if err != nil {
		log.Warn("otel: exporter init failed; traces disabled", zap.Error(err))
		return noop
	}

	res, rerr := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(env),
		),
	)
	if rerr != nil {
		log.Warn("otel: resource merge failed; using default", zap.Error(rerr))
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Info("otel: tracing initialized", zap.String("service", serviceName))
	return tp.Shutdown
}
