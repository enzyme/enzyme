package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/enzyme/api/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// Telemetry holds the OTel SDK providers for shutdown.
type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	logProvider    *sdklog.LoggerProvider
}

// Init initializes OpenTelemetry with OTLP exporters based on config.
func Init(cfg config.TelemetryConfig, version string) (*Telemetry, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(version),
		),
		resource.WithHost(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tel := &Telemetry{}
	// Ignore error: we're already returning a construction error to the caller.
	shutdown := func() {
		_ = tel.Shutdown(ctx)
	}

	// Traces
	if cfg.Traces {
		traceExporter, err := newTraceExporter(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("creating trace exporter: %w", err)
		}

		var sampler sdktrace.Sampler
		switch {
		case cfg.SampleRate <= 0:
			sampler = sdktrace.NeverSample()
		case cfg.SampleRate >= 1:
			sampler = sdktrace.AlwaysSample()
		default:
			sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
		}

		tel.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sampler),
		)
		otel.SetTracerProvider(tel.tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}

	// Metrics
	if cfg.Metrics {
		metricExporter, err := newMetricExporter(ctx, cfg)
		if err != nil {
			shutdown()
			return nil, fmt.Errorf("creating metric exporter: %w", err)
		}

		tel.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(60*time.Second))),
		)
		otel.SetMeterProvider(tel.meterProvider)

		if err := runtime.Start(runtime.WithMeterProvider(tel.meterProvider)); err != nil {
			shutdown()
			return nil, fmt.Errorf("starting runtime instrumentation: %w", err)
		}
	}

	// Logs
	if cfg.Logs {
		logExporter, err := newLogExporter(ctx, cfg)
		if err != nil {
			shutdown()
			return nil, fmt.Errorf("creating log exporter: %w", err)
		}

		tel.logProvider = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		)
		global.SetLoggerProvider(tel.logProvider)
	}

	return tel, nil
}

// Noop returns a Telemetry instance that does nothing on Shutdown.
func Noop() *Telemetry {
	return &Telemetry{}
}

// Shutdown flushes and shuts down both providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutting down tracer provider: %w", err))
		}
	}
	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutting down meter provider: %w", err))
		}
	}
	if t.logProvider != nil {
		if err := t.logProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutting down log provider: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown: %w", errors.Join(errs...))
	}
	return nil
}

func newTraceExporter(ctx context.Context, cfg config.TelemetryConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Protocol {
	case "http":
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
		}
		return otlptracehttp.New(ctx, opts...)
	default:
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
		}
		return otlptracegrpc.New(ctx, opts...)
	}
}

func newMetricExporter(ctx context.Context, cfg config.TelemetryConfig) (sdkmetric.Exporter, error) {
	switch cfg.Protocol {
	case "http":
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
		}
		return otlpmetrichttp.New(ctx, opts...)
	default:
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
		}
		return otlpmetricgrpc.New(ctx, opts...)
	}
}

func newLogExporter(ctx context.Context, cfg config.TelemetryConfig) (sdklog.Exporter, error) {
	switch cfg.Protocol {
	case "http":
		opts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
		}
		return otlploghttp.New(ctx, opts...)
	default:
		opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlploggrpc.WithHeaders(cfg.Headers))
		}
		return otlploggrpc.New(ctx, opts...)
	}
}
