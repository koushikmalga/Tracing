package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/koushikmalga/new_grpc1/pb"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const serviceName = "service6"

type server struct {
	pb.UnimplementedGCDServiceServer
}

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
			semconv.ServiceNameKey.String("service6"),
			semconv.ServiceVersionKey.String("v0.1.0"),
		),
	)
	return r
}

func main() {

	lis, err := net.Listen("tcp", ":10010")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	l := log.New(os.Stdout, "", 0)

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
	s := grpc.NewServer(grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))

	pb.RegisterGCDServiceServer(s, &server{})
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) Compute(ctx context.Context, r *pb.GCDRequest) (*pb.GCDResponse, error) {

	fmt.Println(trace.SpanContextFromContext(ctx))

	newctx, span := otel.Tracer(serviceName).Start(ctx, "compute")
	defer span.End()

	a, b := r.A, r.B
	for b != 0 {
		a, b = b, a%b
	}
	d := a
	time.Sleep(time.Millisecond * 10)
	res, err := s.Compute1(newctx, r)
	if err == nil {
		d = d + res.Result
	}
	return &pb.GCDResponse{Result: d}, nil
}

func (s *server) Compute1(ctx context.Context, r *pb.GCDRequest) (*pb.GCDResponse, error) {

	ctx, span := otel.Tracer(serviceName).Start(ctx, "compute1")
	defer span.End()

	a, b := r.A, r.B
	for b != 0 {
		a, b = b, a%b
	}
	time.Sleep(time.Millisecond * 10)
	return &pb.GCDResponse{Result: a}, nil
}
