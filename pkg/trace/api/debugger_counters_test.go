// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDebuggerCountersWrap(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		implicit200 bool // Write without explicit WriteHeader
		wantAcc     int64
		wantNAcc    int64
	}{
		{"200 explicit", 200, false, 1, 0},
		{"202 accepted", 202, false, 1, 0},
		{"204 no content", 204, false, 1, 0},
		{"299 edge", 299, false, 1, 0},
		{"300 not accepted", 300, false, 0, 1},
		{"400 bad request", 400, false, 0, 1},
		{"500 server err", 500, false, 0, 1},
		{"502 bad gateway", 502, false, 0, 1},
		{"implicit 200 via Write", 0, true, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &debuggerCounters{}
			h := c.wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.implicit200 {
					_, _ = w.Write([]byte("ok"))
					return
				}
				w.WriteHeader(tc.status)
			}))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest("POST", "/debugger/v2/input", strings.NewReader("body")))
			rcv, acc, nacc := c.snapshot()
			if rcv != 1 {
				t.Errorf("received = %d, want 1", rcv)
			}
			if acc != tc.wantAcc {
				t.Errorf("accepted = %d, want %d", acc, tc.wantAcc)
			}
			if nacc != tc.wantNAcc {
				t.Errorf("not_accepted = %d, want %d", nacc, tc.wantNAcc)
			}
		})
	}
}

func TestDebuggerCountersPanicCounted(t *testing.T) {
	c := &debuggerCounters{}
	h := c.wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate an outer middleware writing 500 after recover.
		// Our defer runs first (LIFO); to model outer recovery we wrap
		// in a recovering middleware here.
		panic("boom")
	}))
	outer := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	}
	rr := httptest.NewRecorder()
	outer(rr, httptest.NewRequest("POST", "/debugger/v2/input", strings.NewReader("body")))
	rcv, acc, nacc := c.snapshot()
	if rcv != 1 || acc != 0 || nacc != 1 {
		t.Fatalf("got received=%d accepted=%d not_accepted=%d, want 1/0/1", rcv, acc, nacc)
	}
}

func TestIsDebuggerEndpoint(t *testing.T) {
	cases := map[string]bool{
		"/debugger/v1/input":       true,
		"/debugger/v1/diagnostics": true,
		"/debugger/v2/input":       true,
		"/symdb/v1/input":          false,
		"/v0.4/traces":             false,
		"/info":                    false,
		"/debugger":                false,
	}
	for in, want := range cases {
		if got := isDebuggerEndpoint(in); got != want {
			t.Errorf("isDebuggerEndpoint(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDebuggerCountersHandlerJSON(t *testing.T) {
	c := &debuggerCounters{}
	c.received.Store(10)
	c.accepted.Store(7)
	c.notAccepted.Store(2)
	rr := httptest.NewRecorder()
	c.handlerJSON().ServeHTTP(rr, httptest.NewRequest("GET", "/1337.json", nil))
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	var out map[string]int64
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	want := map[string]int64{"received": 10, "accepted": 7, "not_accepted": 2, "in_flight": 1}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("%s = %d, want %d", k, out[k], v)
		}
	}
}

func TestDebuggerCountersHandlerText(t *testing.T) {
	c := &debuggerCounters{}
	c.received.Store(5)
	c.accepted.Store(3)
	c.notAccepted.Store(1)
	rr := httptest.NewRecorder()
	c.handlerText().ServeHTTP(rr, httptest.NewRequest("GET", "/1337", nil))
	body := rr.Body.String()
	for _, want := range []string{"received:     5", "accepted:     3", "not_accepted: 1", "in_flight:    1"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\nfull body:\n%s", want, body)
		}
	}
}
