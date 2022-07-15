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
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
)

const (
	defaultClientSvcAddr = "localhost:8080"
	defaultWorkers       = 20
	defaultConcurrency   = 1
	defaultRounds        = 0
	defaultIntervalMs    = 1000
)

var testCases = []query{
	{"love", 3040},
	{"friend", 1036},
	{"hello", 349},
	{"world", 728},
	{"sweet", 958},
	{"tear", 463},
	{"faith", 484},
	{"to be, or not to be", 1},
	{"what's past is prologue", 1},
	{"insolence", 14},
}

func init() {
	clientSvcAddr := defaultClientSvcAddr
	if os.Getenv("CLIENT_SVC_ADDR") != "" {
		clientSvcAddr = os.Getenv("CLIENT_SVC_ADDR")
	}
	var err error
	reqURL, err = url.Parse(fmt.Sprintf("http://" + clientSvcAddr))
	if err != nil {
		log.Fatalf("failed to build request URL for %v: %v", clientSvcAddr, err)
	}
	numWorkers = defaultWorkers
	if os.Getenv("NUM_WORKERS") != "" {
		w, err := strconv.ParseInt(os.Getenv("NUM_WORKERS"), 10, 64)
		if err != nil {
			log.Fatalf("failed to parse NUM_WORKERS: %v", err)
		}
		numWorkers = int(w)
	}
	numConcurrency = defaultConcurrency
	if os.Getenv("NUM_CONCURRENCY") != "" {
		c, err := strconv.ParseInt(os.Getenv("NUM_CONCURRENCY"), 10, 64)
		if err != nil {
			log.Fatalf("failed to parse NUM_CONCURRENCY: %v", err)
		}
		numConcurrency = int(c)
	}
	numRounds = defaultRounds
	if os.Getenv("NUM_ROUNDS") != "" {
		r, err := strconv.ParseInt(os.Getenv("NUM_ROUNDS"), 10, 64)
		if err != nil {
			log.Fatalf("failed to parse NUM_ROUNDS: %v", err)
		}
		numRounds = int(r)
	}
	intervalMs = defaultIntervalMs
	if os.Getenv("INTERVAL_MS") != "" {
		i, err := strconv.ParseInt(os.Getenv("INTERVAL_MS"), 10, 64)
		if err != nil {
			log.Fatalf("failed to parse INTERVAL_MS: %v", err)
		}
		intervalMs = int(i)
	}
}
