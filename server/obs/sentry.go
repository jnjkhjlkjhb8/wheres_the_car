package obs

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Init(service string) func() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Printf("[SENTRY] action=init event=disabled service=%s reason=no_dsn", service)
		return func() {}
	}
	tracesRate := 0.1
	if v := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			tracesRate = f
		}
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
		ServerName:       service,
		EnableTracing:    tracesRate > 0,
		TracesSampleRate: tracesRate,
	})
	if err != nil {
		log.Printf("[SENTRY] action=init event=failed service=%s error=%v", service, err)
		return func() {}
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("service", service)
	})
	log.SetOutput(captureWriter{out: os.Stderr})
	log.Printf("[SENTRY] action=init event=enabled service=%s traces=%.2f", service, tracesRate)
	return func() { sentry.Flush(2 * time.Second) }
}

type captureWriter struct{ out io.Writer }

func (w captureWriter) Write(p []byte) (int, error) {
	if msg, ok := errorLogLine(string(p)); ok {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetLevel(sentry.LevelError)
		hub.Scope().SetTag("source", "log")
		hub.CaptureMessage(msg)
	}
	return w.out.Write(p)
}

func errorLogLine(line string) (string, bool) {
	i := strings.LastIndex(line, "error=")
	if i < 0 {
		return "", false
	}
	val := strings.TrimSpace(line[i+len("error="):])
	if val == "" || strings.HasPrefix(val, "<nil>") || strings.HasPrefix(val, "%!") {
		return "", false
	}
	return strings.TrimSpace(line), true
}

func Recover(name string) {
	if r := recover(); r != nil {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("job", name)
		hub.RecoverWithContext(context.Background(), r)
		hub.Flush(2 * time.Second)
		panic(r)
	}
}

func Capture(name string, err error) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTag("job", name)
	hub.CaptureException(err)
}

func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("grpc.method", info.FullMethod)
		ctx = sentry.SetHubOnContext(ctx, hub)
		defer func() {
			if r := recover(); r != nil {
				hub.RecoverWithContext(ctx, r)
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		resp, err := handler(ctx, req)
		if shouldReport(err) {
			hub.CaptureException(err)
		}
		return resp, err
	}
}

func StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("grpc.method", info.FullMethod)
		defer func() {
			if r := recover(); r != nil {
				hub.RecoverWithContext(ss.Context(), r)
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		err = handler(srv, ss)
		if shouldReport(err) {
			hub.CaptureException(err)
		}
		return err
	}
}

func shouldReport(err error) bool {
	if err == nil {
		return false
	}
	switch status.Code(err) {
	case codes.Unknown, codes.Internal, codes.Unavailable, codes.DataLoss, codes.Unimplemented:
		return true
	default:
		return false
	}
}
