package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

const serviceName = "service2"

var fla = 0

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

	f1, err1 := os.OpenFile("/Traces/finaltrace1.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err1 != nil {
		os.Create("/Traces/finaltrace1.txt")
		f1, err1 = os.OpenFile("/Traces/finaltrace1.txt", os.O_APPEND|os.O_WRONLY, 0600)
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
		if err := tp.Shutdown(ctx); err != nil {
			l.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	Handler := func(w http.ResponseWriter, req *http.Request) {
		ctx = req.Context()

		ctx, span := otel.Tracer(serviceName).Start(ctx, "service25")

		if fla == 0 {
			fmt.Println("hello")
			fla++
			ctx1, span1 := otel.Tracer(serviceName).Start(ctx, "service2->service3")

			client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

			req1, _ := http.NewRequestWithContext(ctx1, "GET", "http://sidecar1:9093/", nil)
			Resp1, err := client.Do(req1)
			if err != nil {
				fmt.Println(err)
				span1.SetStatus(codes.Error, "http request has failed")
			}

			Resp1.Body.Close()
			span1.End()

			ctx2, span2 := otel.Tracer(serviceName).Start(ctx, "service2->service4")

			client1 := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

			req2, _ := http.NewRequestWithContext(ctx2, "GET", "http://final4:9094/", nil)
			Resp2, err := client1.Do(req2)
			if err != nil {
				fmt.Println(err)
				span2.SetStatus(codes.Error, "http request has failed")
			}

			defer Resp2.Body.Close()
			span2.End()

		} else {
			_, span1 := otel.Tracer(serviceName).Start(ctx, "service2->service3")

			go func() {

				client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
				ctx3 := context.Background()
				ctx3 = trace.ContextWithSpan(ctx3, span1)

				req1, _ := http.NewRequestWithContext(ctx3, "GET", "http://sidecar1:9093/", nil)
				Resp1, err := client.Do(req1)
				if err != nil {
					fmt.Println(err)
					span1.SetStatus(codes.Error, "http request has failed")
				}

				defer Resp1.Body.Close()
			}()
			span1.End()

			_, span2 := otel.Tracer(serviceName).Start(ctx, "service2->service4")

			go func() {
				// url1 := flag.String("server1", "http://service4:9094/gcd/125/125", "server url1")
				// flag.Parse()

				client1 := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
				ctx4 := context.Background()
				ctx4 = trace.ContextWithSpan(ctx4, span2)

				req2, _ := http.NewRequestWithContext(ctx4, "GET", "http://final4:9094/", nil)
				Resp2, err := client1.Do(req2)
				if err != nil {
					fmt.Println(err)
					span2.SetStatus(codes.Error, "http request has failed")
				}

				defer Resp2.Body.Close()
			}()
			span2.End()

			fla = 0
		}
		defer span.End()
	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(Handler), "/")
	http.Handle("/", otelHandler)

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":10010", nil))
}

type stop struct {
	error
}
