# Cloud Trace and Cloud Profiler codelab in Go

These are the resource files needed for the Cloud Trace and Cloud Profiler codelab from Google.

## What you'll build

In this codelab, you will instrument distributed traces to HTTP/gRPC clients/servers and
instrument a profiler agent in a server.

## What you'll learn

- How to get started with the [OpenTelemetry](https://opentelemetry.io) Trace libraries in Go project
- How to create a span with the library
- How to propagate span contexts across the wire between app components
- How to send trace data to [Cloud Trace](https://cloud.google.com/trace/docs)
- How to analyze the trace on Cloud Trace
- How to embed profiler agent
- How to investigate the bottle neck on [Cloud Profiler](https://cloud.google.com/profiler/docs)

## Folders

The folders in this repository contains the materials for multiple codelab.

- Part 1: trace
  - step 0: starting point of the codelab
  - step 1: finish trace instrumentation between loadgen and client with stdout exporter
    - HTTP instrumentation
  - step 2: finish trace instrumentation between client and server with stdout exporter
    - gRPC instrumentation
  - step 3: replace stdout exporters with Cloud Trace exporters
  - step 4: add more spans in server
- Part 2: profile
  - step 5: add profiler in server
  - step 6: tune up the server
