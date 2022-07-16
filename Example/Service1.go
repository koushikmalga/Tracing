package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

const serviceName = "Service1"

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
		os.Create("/Traces/finaltrace.txt")
		f1, err1 = os.OpenFile("/Traces/finaltrace.txt", os.O_APPEND|os.O_WRONLY, 0600)
	}

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

	url := flag.String("server", "http://final2:9092/", "server url")
	flag.Parse()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	router := httprouter.New()

	router.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ctx, span := otel.Tracer(serviceName).Start(ctx, "service1")
		// fmt.Println(span.SpanContext())
		req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)
		Resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			span.SetStatus(codes.Error, "http request has failed")
		}

		defer Resp.Body.Close()
		span.End()

	})

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":10010", router))
}

type stop struct {
	error
}
