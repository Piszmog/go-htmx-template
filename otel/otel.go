package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

type Config struct {
	Insecure    bool
	Endpoint    string
	ServiceName string
}

type Shutdown func(ctx context.Context) error

func Init(ctx context.Context, config Config) (Shutdown, error) {
	traceShutdown, err := initTrace(ctx, config)
	if err != nil {
		return nil, err
	}

	metricShutdown, err := initMetrics(ctx, config)
	if err != nil {
		return nil, err
	}

	shutdownFunc := func(ctx context.Context) error {
		if err := traceShutdown(ctx); err != nil {
			return err
		}
		if err := metricShutdown(ctx); err != nil {
			return err
		}
		return nil
	}

	return shutdownFunc, nil
}

func initTrace(ctx context.Context, config Config) (Shutdown, error) {
	shutdownFunc := func(ctx context.Context) error { return nil }

	switch config.Endpoint {
	case "":
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
	default:
		options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(config.Endpoint)}
		if config.Insecure {
			options = append(options, otlptracegrpc.WithInsecure())
		}

		exporter, err := otlptracegrpc.New(ctx, options...)
		if err != nil {
			return nil, err
		}

		tp := trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(
				resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(config.ServiceName),
				),
			),
		)
		shutdownFunc = func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		}
		otel.SetTracerProvider(tp)
	}

	return shutdownFunc, nil
}

func initMetrics(ctx context.Context, config Config) (Shutdown, error) {
	shudownFunc := func(context.Context) error { return nil }

	switch config.Endpoint {
	case "":
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
	default:
		options := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(config.Endpoint)}
		if config.Insecure {
			options = append(options, otlpmetricgrpc.WithInsecure())
		}
		exporter, err := otlpmetricgrpc.New(ctx, options...)
		if err != nil {
			return nil, err
		}

		mp := metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(exporter)),
			metric.WithResource(
				resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(config.ServiceName),
				),
			),
		)
		otel.SetMeterProvider(mp)
		shudownFunc = func(ctx context.Context) error {
			return mp.Shutdown(ctx)
		}
	}

	return shudownFunc, nil
}
