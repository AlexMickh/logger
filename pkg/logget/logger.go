package logger

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type key string

var (
	Key       = key("logger")
	RequestID = "request_id"
)

type Logger struct {
	log *zap.Logger
}

func New(ctx context.Context, env string) (context.Context, error) {
	var logger *zap.Logger
	var err error

	switch env {
	case "prod":
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	case "dev":
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	default:
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	return context.WithValue(ctx, Key, &Logger{log: logger}), nil
}

func GetFromCtx(ctx context.Context) *Logger {
	return ctx.Value(Key).(*Logger)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Info(msg, fields...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Fatal(msg, fields...)
}

func (l *Logger) With(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, Key, &Logger{log: l.log.With(fields...)})
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}
	l.log.Error(msg, fields...)
}

func Interceptor(ctx context.Context) grpc.UnaryServerInterceptor {
	return func(lCtx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log := GetFromCtx(ctx)
		lCtx = context.WithValue(lCtx, Key, log)

		md, ok := metadata.FromIncomingContext(lCtx)
		if ok {
			guid, ok := md[RequestID]
			if ok {
				GetFromCtx(lCtx).Error(ctx, "No request id")
				ctx = context.WithValue(ctx, RequestID, guid)
			}
		}

		GetFromCtx(lCtx).Info(lCtx, "request",
			zap.String("method", info.FullMethod),
			zap.Time("request time", time.Now()),
		)

		return handler(lCtx, req)
	}
}

func ChiMiddleware(ctx context.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		ctx = GetFromCtx(ctx).With(ctx,
			zap.String("component", "middleware/logger"),
		)

		GetFromCtx(ctx).Info(ctx, "logger middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := GetFromCtx(ctx).With(ctx,
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				GetFromCtx(entry).Info(ctx, "request completed",
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.String("duration", time.Since(t1).String()),
				)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
