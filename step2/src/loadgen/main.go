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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	reqURL *url.URL

	// All configuration numbers can be tweaked via manifest file
	numWorkers     int
	numConcurrency int
	numRounds      int
	intervalMs     int

	// step1. setup customized HTTP client
	httpClient = http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
)

type query struct {
	query     string
	wantCount int
}

// step1. add OpenTelemetry initialization function
func initTracer() (*sdktrace.TracerProvider, error) {
	// create a stdout exporter to show collected spans out to stdout.
	exporter, err := stdout.New(stdout.WithPrettyPrint())
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

func main() {
	// step1. setup OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize TracerProvider: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down TracerProvider: %v", err)
		}
	}()
	// step1. end setup

	log.Printf("starting worder with %d workers in %d concurrency", numWorkers, numConcurrency)
	log.Printf("number of rounds: %d (0 is inifinite)", numRounds)

	t := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	i := 0
	for range t.C {
		log.Printf("simulating client requests, round %d", i)
		if err := run(numWorkers, numConcurrency); err != nil {
			log.Printf("aborted round with error: %v", err)
		}
		log.Printf("simulated %d requests", numWorkers)
		if numRounds != 0 && i > numRounds {
			break
		}
		i++
	}
}

// run is the worker generator in concurrent.
func run(workers, concurrency int) error {
	respErrCh := make(chan error)
	concCh := make(chan bool, concurrency)
	for n := 0; n < workers; n++ {
		go func() {
			concCh <- true
			defer func() {
				<-concCh
			}()
			respErrCh <- func() error {
				q := testCases[rand.Intn(len(testCases))]
				matched, err := runQuery(q.query)
				if err != nil {
					return err
				}
				check(q, matched)
				return nil
			}()
		}()
	}

	for i := 0; i < workers; i++ {
		if err := <-respErrCh; err != nil {
			return err
		}
	}
	return nil
}

// runQuery throws a query s to the client and returns the number of matched line results
//
// TODO: instrument this method to trace all requests down to the server.
func runQuery(s string) (int, error) {
	v := url.Values{}
	v.Set("q", s)
	reqURL.RawQuery = v.Encode()

	// step1. instrument trace
	ctx := context.Background()
	tr := otel.Tracer("loadgen")
	ctx, span := tr.Start(ctx, "query.request", trace.WithAttributes(
		semconv.TelemetrySDKLanguageGo,
		semconv.ServiceNameKey.String("loadgen.runQuery"),
		attribute.Key("query").String(s),
	))
	defer span.End()
	ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return -1, fmt.Errorf("error creating HTTP request object: %v", err)
	}
	resp, err := httpClient.Do(req)
	// step1. end instrumentation
	if err != nil {
		return -1, fmt.Errorf("error sending request to %v: %v", reqURL.String(), err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, fmt.Errorf("error reading response body: %v", err)
	}
	r := struct {
		Matched int `json:"match_count"`
	}{}
	if err = json.Unmarshal(data, &r); err != nil {
		return -1, err
	}
	return r.Matched, nil
}

// check compares expected counts of the query word and matched count
func check(q query, matched int) {
	if q.wantCount != matched {
		log.Printf("query '%s' had issue: expected %d, matched %d", q.query, q.wantCount, matched)
		return
	}
	log.Printf("query '%s': matched %d", q.query, matched)
}
