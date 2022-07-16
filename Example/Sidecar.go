package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

const serviceName = "sidecarservice"

var ctx = context.Background()

func newExporter(w io.Writer) (tracesdk.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// // Do not print timestamps for the demo.
		// stdouttrace.WithoutTimestamps(),
	)
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("v0.1.0"),
		),
	)
	return r
}

func main() {

	l := log.New(os.Stdout, "", 0)

	// Write telemetry data to a file.
	f1, err1 := os.OpenFile("/Traces/finaltrace.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err1 != nil {
		l.Fatal(err1)
	}
	defer f1.Close()

	exp, err := newExporter(f1)
	if err != nil {
		l.Fatal(err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	Handler := func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()

		_, span := otel.Tracer("sidecar").Start(ctx, "service3")

		req, err := http.NewRequest("GET", "http://localhost:10010/", nil)

		if err != nil {
			panic(err)
		}

		Resp, err := http.DefaultClient.Do(req)

		if err != nil {
			fmt.Println(err)
			span.SetStatus(codes.Error, "http request has failed")
		}

		defer Resp.Body.Close()
		defer span.End()

	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(Handler), "/")
	http.Handle("/", otelHandler)

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":10011", nil))
}

type stop struct {
	error
}
