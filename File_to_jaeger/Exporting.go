package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type Code uint32

const (
	// Unset is the default status code.
	Unset Code = 0
	// Error indicates the operation contains an error.
	Error Code = 1
	// Ok indicates operation has been validated by an Application developers
	// or Operator to have completed successfully, or contain no error.
	Ok Code = 2

	maxCode = 3
)

type Val1 struct {
	Type  string
	Value interface{}
}
type KeyValue1 struct {
	Key   string
	Value Val1
}

// converting string to Code.
var strToCode = map[string]Code{
	`"Unset"`: Unset,
	`"Error"`: Error,
	`"Ok"`:    Ok,
}

type SpanContext1 struct {
	TraceID    string
	SpanID     string
	TraceFlags string
	TraceState string
	Remote     bool
}

type Event1 struct {
	Name                  string
	Attributes            []KeyValue1
	DroppedAttributeCount int
	Time                  time.Time
}

type Link1 struct {
	SpanContext           SpanContext1
	Attributes            []KeyValue1
	DroppedAttributeCount int
}

type Status1 struct {
	Code        string
	Description string
}

type Library1 struct {
	Name      string
	Version   string
	SchemaURL string
}
type SpanStub struct {
	Name                   string
	SpanContext            SpanContext1
	Parent                 SpanContext1
	SpanKind               int
	StartTime              time.Time
	EndTime                time.Time
	Attributes             []KeyValue1
	Events                 []Event1
	Links                  []Link1
	Status                 Status1
	DroppedAttributes      int
	DroppedEvents          int
	DroppedLinks           int
	ChildSpanCount         int
	Resource               []KeyValue1
	InstrumentationLibrary Library1
}

type Span struct {
	Name                   string
	SpanContext            trace.SpanContext
	Parent                 trace.SpanContext
	SpanKind               trace.SpanKind
	StartTime              time.Time
	EndTime                time.Time
	Attributes             []attribute.KeyValue
	Events                 []tracesdk.Event
	Links                  []tracesdk.Link
	Status                 tracesdk.Status
	DroppedAttributes      int
	DroppedEvents          int
	DroppedLinks           int
	ChildSpanCount         int
	Resource               *resource.Resource
	InstrumentationLibrary instrumentation.Library
}

func attr(kv []KeyValue1) []attribute.KeyValue {
	//This function is to convert attributes which are in []KeyValue1 to attribute.KeyValue
	var att []attribute.KeyValue
	for i := range kv {
		switch kv[i].Value.Type {
		case "BOOL":
			att = append(att, attribute.Bool(kv[i].Key, kv[i].Value.Value.(bool)))
		case "BOOLSLICE":
			att = append(att, attribute.BoolSlice(kv[i].Key, kv[i].Value.Value.([]bool)))
		case "INT64":
			switch kv[i].Value.Value.(type) {
			case float64:
				k1 := int64(kv[i].Value.Value.(float64))
				att = append(att, attribute.Int64(kv[i].Key, k1))
			default:
				att = append(att, attribute.Int64(kv[i].Key, kv[i].Value.Value.(int64)))
			}
		case "INT64SLICE":
			att = append(att, attribute.Int64Slice(kv[i].Key, kv[i].Value.Value.([]int64)))
		case "FLOAT64":
			att = append(att, attribute.Float64(kv[i].Key, kv[i].Value.Value.(float64)))
		case "FLOAT64SLICE":
			att = append(att, attribute.Float64Slice(kv[i].Key, kv[i].Value.Value.([]float64)))
		case "STRING":
			att = append(att, attribute.String(kv[i].Key, kv[i].Value.Value.(string)))
		case "STRINGSLICE":
			att = append(att, attribute.StringSlice(kv[i].Key, kv[i].Value.Value.([]string)))

		}
	}
	return att
}

func Eve(ev []Event1) []tracesdk.Event {
	//converting []Event1 to tracesdk.Event format.
	var k []tracesdk.Event
	for i := range ev {
		var k2 tracesdk.Event
		k1 := attr(ev[i].Attributes)
		k2.Name = ev[i].Name
		k2.Attributes = k1
		k2.DroppedAttributeCount = ev[i].DroppedAttributeCount
		k2.Time = ev[i].Time
		k = append(k, k2)
	}
	return k
}

func convContext(sp SpanContext1) trace.SpanContext {
	//converting SpanContext which is in SpanContext1 format to trace.SpanContext
	traceID, err := trace.TraceIDFromHex(sp.TraceID)
	if err != nil {
		panic(err)
	}
	spanID, err := trace.SpanIDFromHex(sp.SpanID)
	if err != nil {
		panic(err)
	}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  sp.Remote,
	})
	return spanContext
}

func Lin(li []Link1) []tracesdk.Link {
	//converting Links which is in []Link1 format to []tracesdk.Link
	var k []tracesdk.Link
	for i := range li {
		var k2 tracesdk.Link
		k1 := attr(li[i].Attributes)
		k2.Attributes = k1
		k2.DroppedAttributeCount = li[i].DroppedAttributeCount
		k2.SpanContext = convContext(li[i].SpanContext)
		k = append(k, k2)
	}
	return k
}

func convStatus(st Status1) tracesdk.Status {
	//Converting Status which is in Status1 format to tracesd.Status
	var k tracesdk.Status
	k.Description = st.Description
	k.Code = codes.Code(strToCode[st.Code])

	return k
}

func convLib(sil Library1) instrumentation.Library {
	var k instrumentation.Library
	k.Name = sil.Name
	k.Version = sil.Version
	k.SchemaURL = sil.SchemaURL
	return k
}

func convert(sp SpanStub) Span {
	var spa Span
	spa.Name = sp.Name
	spa.SpanContext = convContext(sp.SpanContext)
	// Checking whether there is a parent Trace Id
	if sp.Parent.TraceID != "00000000000000000000000000000000" {
		spa.Parent = convContext(sp.Parent)
	}
	spa.SpanKind = trace.SpanKind(sp.SpanKind)
	spa.StartTime = sp.StartTime
	spa.EndTime = sp.EndTime
	spa.Attributes = attr(sp.Attributes)
	spa.Events = Eve(sp.Events)
	spa.Links = Lin(sp.Links)
	spa.Status = convStatus(sp.Status)
	spa.DroppedAttributes = sp.DroppedAttributes
	spa.DroppedEvents = sp.DroppedEvents
	spa.DroppedLinks = sp.DroppedLinks
	spa.ChildSpanCount = sp.ChildSpanCount
	k1 := attr(sp.Resource)
	r := resource.NewSchemaless(k1[0])
	for i := range k1 {
		if i != 0 {
			r, _ = resource.Merge(r, resource.NewSchemaless(k1[i]))
		}
	}
	spa.Resource = r
	spa.InstrumentationLibrary = convLib(sp.InstrumentationLibrary)
	return spa
}

func main() {
	// Let's read the traces file.
	content, err := ioutil.ReadFile("/Traces/finaltrace1.txt")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	//converting traces.txt to string format
	content1 := string(content)
	f := strings.NewReader(content1)
	// creating a decoder
	dec := json.NewDecoder(f)
	var dat []SpanStub
	i := 0
	for {

		var data SpanStub
		// decoding each span into data variable
		if err := dec.Decode(&data); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		dat = append(dat, data)
		i++
	}
	// Creating array of SpanStubs of length dat
	s := make(tracetest.SpanStubs, len(dat))
	for i := range dat {
		//converting data to span foormat.
		data1 := convert(dat[i])
		if dat[i].Parent.TraceID == "00000000000000000000000000000000" {
			s[i] = tracetest.SpanStub{
				Name:                   data1.Name,
				SpanContext:            data1.SpanContext,
				SpanKind:               data1.SpanKind,
				StartTime:              data1.StartTime,
				EndTime:                data1.EndTime,
				Attributes:             data1.Attributes,
				Links:                  data1.Links,
				DroppedAttributes:      data1.DroppedAttributes,
				DroppedEvents:          data1.DroppedEvents,
				DroppedLinks:           data1.DroppedLinks,
				ChildSpanCount:         data1.ChildSpanCount,
				Resource:               data1.Resource,
				InstrumentationLibrary: data1.InstrumentationLibrary,
			}
		} else {
			s[i] = tracetest.SpanStub{
				Name:                   data1.Name,
				SpanContext:            data1.SpanContext,
				Parent:                 data1.Parent,
				SpanKind:               data1.SpanKind,
				StartTime:              data1.StartTime,
				EndTime:                data1.EndTime,
				Attributes:             data1.Attributes,
				Links:                  data1.Links,
				DroppedAttributes:      data1.DroppedAttributes,
				DroppedEvents:          data1.DroppedEvents,
				DroppedLinks:           data1.DroppedLinks,
				ChildSpanCount:         data1.ChildSpanCount,
				Resource:               data1.Resource,
				InstrumentationLibrary: data1.InstrumentationLibrary,
			}
		}

	}

	ctx := context.Background()
	//exporting spans which we converted to jaeger collector.
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://simplest-collector:14268/api/traces")))
	if err == nil {
		exp.ExportSpans(ctx, s.Snapshots())
	}

}

