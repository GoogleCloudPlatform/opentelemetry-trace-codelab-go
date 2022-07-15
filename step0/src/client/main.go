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
	"net/http"
	"net/url"
	"os"
	"time"

	"opentelemetry-trace-codelab-go/client/shakesapp"

	"google.golang.org/grpc"
)

const (
	listenPort = "8080"
)

type clientService struct {
	serverSvcAddr string
	serverSvcConn *grpc.ClientConn
}

func NewClientService() *clientService {
	return &clientService{}
}

// handler accepts HTTP requests from the loadgen and pass the query down to the server.
//
// TODO: instrument this method to trace the request down to the server.
func (cs *clientService) handler(w http.ResponseWriter, r *http.Request) {
	// NOTE: do not pass the raw query in produxtion systems.
	rawQuery := r.URL.Query().Get("q")
	query, err := url.QueryUnescape(rawQuery)
	if err != nil {
		writeError(w, fmt.Sprintf("can't unescape the query: %s", rawQuery))
		return
	}

	ctx := r.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cli := shakesapp.NewShakespeareServiceClient(cs.serverSvcConn)
	resp, err := cli.GetMatchCount(ctx, &shakesapp.ShakespeareRequest{
		Query: query,
	})
	if err != nil {
		writeError(w, fmt.Sprintf("error calling GetMatchCount: %v", err))
		return
	}
	ret, err := json.Marshal(resp)
	if err != nil {
		writeError(w, fmt.Sprintf("error marshalling data: %v", err))
		return
	}
	log.Println(string(ret))
	if _, err = w.Write(ret); err != nil {
		writeError(w, fmt.Sprintf("error on writing response: %v", err))
		return
	}
}

// health is the health check handler.
func (cs *clientService) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {
	ctx := context.Background()
	svc := NewClientService()
	mustMapEnv(&svc.serverSvcAddr, "SERVER_SVC_ADDR")
	mustConnGRPC(ctx, &svc.serverSvcConn, svc.serverSvcAddr)

	http.HandleFunc("/", svc.handler)
	http.HandleFunc("/_genki", svc.health)

	port := listenPort
	if os.Getenv("CLIENT_PORT") != "" {
		port = os.Getenv("CLIENT_PORT")
	}
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil); err != nil {
		log.Fatalf("error listening HTTP server: %v", err)
	}
}

// mustMapEnv assigns the value of environment variable envKey to target.
func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		log.Fatalf("environment variable %q not set", envKey)
	}
	*target = v
}

// Helper function for gRPC connections: Dial and create client once, reuse.
func mustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error
	*conn, err = grpc.DialContext(ctx, addr,
		grpc.WithInsecure(),
		grpc.WithTimeout(time.Second*3),
	)
	if err != nil {
		panic(fmt.Sprintf("Error %s grpc: failed to connect %s", err, addr))
	}
}

// writeError writes error message s to w.
// This function is just for demo use and can't be used in production, because
// it doesn't handle escaping double quote and new lines.
func writeError(w io.Writer, s string) {
	log.Println(s)
	w.Write([]byte(`{"error": "` + s + `"}`))
}
