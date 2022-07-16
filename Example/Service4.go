package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/koushikmalga/new_grpc1/pb"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const serviceName = "service4"

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

		newctx, span := otel.Tracer(serviceName).Start(ctx, "service4")
		defer span.End()
		conn, err := grpc.Dial("final5:9095", grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
			grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()))
		if err != nil {
			log.Fatalf("Dial failed: %v", err)
		}
		gcdClient := pb.NewGCDServiceClient(conn)

		// Parse parameters

		// Call GCD service
		req := &pb.GCDRequest{A: 125, B: 125}
		newctx = metadata.NewOutgoingContext(newctx, metadata.Pairs(
			"timestamp", time.Now().Format(time.StampNano),
			"client-id", "web-api-client",
			"user-id", "test-user",
		))
		if res, err := gcdClient.Compute(newctx, req); err == nil {
			fmt.Println("Result :", res.Result)
		} else {
			panic(fmt.Sprintf("ERROR: %v \n", err.Error()))
		}

	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(Handler), "/")
	http.Handle("/", otelHandler)

	fmt.Println("Running...")
	log.Fatal(http.ListenAndServe(":10010", nil))
}
