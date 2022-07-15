// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"

	"opentelemetry-trace-codelab-go/server/shakesapp"

	"cloud.google.com/go/profiler"
	"cloud.google.com/go/storage"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	listenPort = "5050"

	bucketName   = "dataflow-samples"
	bucketPrefix = "shakespeare/"
)

type serverService struct {
	shakesapp.UnimplementedShakespeareServiceServer
	healthpb.UnimplementedHealthServer
}

func NewServerService() *serverService {
	return &serverService{}
}

// step2. add OpenTelemetry initialization function
func initTracer() (*sdktrace.TracerProvider, error) {
	// step3. replace stdout exporter with Cloud Trace exporter
	// cloudtrace.New() finds the credentials to Cloud Trace automatically following the
	// rules defined by golang.org/x/oauth2/google.findDefaultCredentailsWithParams.
	// https://pkg.go.dev/golang.org/x/oauth2/google#FindDefaultCredentialsWithParams
	exporter, err := cloudtrace.New()
	// step3. end replacing exporter
	if err != nil {
		return nil, err
	}
	// for the demonstration, we use AlwaysSmaple sampler to take all spans.
	// do not use this option in production.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp, nil
}

// step2: end OpenTelemetry initialization function

// step5: add Profiler initializer
func initProfiler() {
	cfg := profiler.Config{
		Service:              "server",
		ServiceVersion:       "1.1.0", // step6. update version
		NoHeapProfiling:      true,
		NoAllocProfiling:     true,
		NoGoroutineProfiling: true,
		NoCPUProfiling:       false,
	}
	if err := profiler.Start(cfg); err != nil {
		log.Fatalf("failed to launch profiler agent: %v", err)
	}
}

// step5: end Profiler initializer

// TODO: instrument the application with Cloud Profiler agent
func main() {
	port := listenPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("error %v; error listening port %v", err, port)
	}

	// step2. setup OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize TracerProvider: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down TracerProvider: %v", err)
		}
	}()
	// step2. end setup

	// step5. start profiler
	go initProfiler()
	// step5. end

	svc := NewServerService()
	// step2: add interceptor
	interceptorOpt := otelgrpc.WithTracerProvider(otel.GetTracerProvider())
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(interceptorOpt)),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(interceptorOpt)),
	)
	// step2: end adding interceptor
	shakesapp.RegisterShakespeareServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("error serving server: %v", err)
	}
}

// GetMatchCount implements a server for ShakespeareService.
//
// TODO: instrument the application to take the latency of the request to Cloud Storage
func (s *serverService) GetMatchCount(ctx context.Context, req *shakesapp.ShakespeareRequest) (*shakesapp.ShakespeareResponse, error) {
	resp := &shakesapp.ShakespeareResponse{}
	texts, err := readFiles(ctx, bucketName, bucketPrefix)
	if err != nil {
		return resp, fmt.Errorf("fails to read files: %s", err)
	}

	// step6. considered the process carefully and naively tuned up by extracting
	// regexp pattern compile process out of for loop.
	query := strings.ToLower(req.Query)
	re := regexp.MustCompile(query)
	for _, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			line = strings.ToLower(line)
			isMatch := re.MatchString(line)
			// step6. done replacing regexp with strings
			if isMatch {
				resp.MatchCount++
			}
		}
	}
	return resp, nil
}

// readFiles reads the content of files within the specified bucket with the
// specified prefix path in parallel and returns their content. It fails if
// operations to find or read any of the files fails.
func readFiles(ctx context.Context, bucketName, prefix string) ([]string, error) {
	type resp struct {
		s   string
		err error
	}

	// step4: add an extra span
	span := trace.SpanFromContext(ctx)
	span.SetName("server.readFiles")
	span.SetAttributes(attribute.Key("bucketname").String(bucketName))
	defer span.End()
	// step4: end add span

	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return []string{}, fmt.Errorf("failed to create storage client: %s", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	var paths []string
	it := bucket.Objects(ctx, &storage.Query{Prefix: bucketPrefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("failed to iterate over files in %s starting with %s: %v", bucketName, prefix, err)
		}
		if attrs.Name != "" {
			paths = append(paths, attrs.Name)
		}
	}

	resps := make(chan resp)
	for _, path := range paths {
		go func(path string) {
			obj := bucket.Object(path)
			r, err := obj.NewReader(ctx)
			if err != nil {
				resps <- resp{"", err}
			}
			defer r.Close()
			data, err := ioutil.ReadAll(r)
			resps <- resp{string(data), err}
		}(path)
	}
	ret := make([]string, len(paths))
	for i := 0; i < len(paths); i++ {
		r := <-resps
		if r.err != nil {
			err = r.err
		}
		ret[i] = r.s
	}
	return ret, err
}

// Check is for health checking.
func (s *serverService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// Watch is for health checking.
func (s *serverService) Watch(req *healthpb.HealthCheckRequest, server healthpb.Health_WatchServer) error {
	return nil
}
