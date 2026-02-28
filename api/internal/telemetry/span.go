package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Pre-computed span start option to avoid per-call allocation.
var dbSpanAttrs = trace.WithAttributes(attribute.String("db.system", "sqlite"))

// StartDBSpan starts a new span for a database operation. The caller must
// call the returned end function when the operation completes, passing the
// error (if any) so the span records failures.
//
//	ctx, end := telemetry.StartDBSpan(ctx, "message.Create")
//	result, err := doQuery(ctx)
//	end(err)
//	return result, err
func StartDBSpan(ctx context.Context, operation string) (context.Context, func(error)) {
	ctx, span := otel.Tracer("enzyme.database").Start(ctx, operation, dbSpanAttrs)
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}
