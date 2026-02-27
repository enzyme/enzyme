package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const dbTracerName = "enzyme.database"

// StartDBSpan starts a new span for a database operation. The caller must
// call the returned end function when the operation completes.
//
//	ctx, end := telemetry.StartDBSpan(ctx, "message.Create")
//	defer end()
func StartDBSpan(ctx context.Context, operation string) (context.Context, func()) {
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, operation,
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
		),
	)
	return ctx, func() { span.End() }
}
