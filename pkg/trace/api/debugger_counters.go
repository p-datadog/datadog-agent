// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

// debuggerCounters tracks payload counts for the trace-agent debugger proxy
// endpoints (/debugger/v1/input, /debugger/v1/diagnostics, /debugger/v2/input).
//
// received increments at handler entry. accepted/notAccepted increment in a
// defer that inspects the captured response status, so a non-2xx written by
// any layer (middleware rejection, ReverseProxy ErrorHandler, intake response
// passthrough) is counted correctly.
//
// In-flight is implicit: received - accepted - notAccepted.
type debuggerCounters struct {
	received    atomic.Int64
	accepted    atomic.Int64
	notAccepted atomic.Int64
}

func (c *debuggerCounters) snapshot() (received, accepted, notAccepted int64) {
	return c.received.Load(), c.accepted.Load(), c.notAccepted.Load()
}

// wrap returns an http.Handler counting requests routed to h. A panic in h is
// counted as not_accepted and re-raised so outer recovery middleware can still
// handle it; without the inline recover, the deferred status check would fire
// before any outer middleware writes a non-2xx response and would miscount the
// request as accepted.
func (c *debuggerCounters) wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.received.Add(1)
		sw := &statusCapturingWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			rec := recover()
			accepted := rec == nil && sw.status >= 200 && sw.status < 300
			if accepted {
				c.accepted.Add(1)
			} else {
				c.notAccepted.Add(1)
			}
			if rec != nil {
				panic(rec)
			}
		}()
		h.ServeHTTP(sw, r)
	})
}

// statusCapturingWriter records the first HTTP status code written to the
// response. Subsequent WriteHeader calls are passed through unchanged but do
// not overwrite the captured status (matching net/http semantics: only the
// first WriteHeader takes effect).
type statusCapturingWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write implicitly causes a WriteHeader(200) per net/http semantics; ensure
// the captured status reflects that if no explicit WriteHeader was made.
func (w *statusCapturingWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush is implemented by httputil.ReverseProxy's response handling; pass
// through to the underlying writer when supported.
func (w *statusCapturingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// handlerText serves the human-readable diagnostic endpoint (/1337).
func (c *debuggerCounters) handlerText() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rcv, acc, nacc := c.snapshot()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "debugger payloads\n")
		fmt.Fprintf(w, "  received:     %d\n", rcv)
		fmt.Fprintf(w, "  accepted:     %d\n", acc)
		fmt.Fprintf(w, "  not_accepted: %d\n", nacc)
		fmt.Fprintf(w, "  in_flight:    %d\n", rcv-acc-nacc)
	})
}

// handlerJSON serves the machine-readable diagnostic endpoint (/1337.json).
func (c *debuggerCounters) handlerJSON() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rcv, acc, nacc := c.snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int64{
			"received":     rcv,
			"accepted":     acc,
			"not_accepted": nacc,
			"in_flight":    rcv - acc - nacc,
		})
	})
}

// isDebuggerEndpoint reports whether the endpoint pattern p refers to a
// trace-agent debugger proxy endpoint that should be counted.
func isDebuggerEndpoint(p string) bool {
	return strings.HasPrefix(p, "/debugger/")
}

// DebuggerCountersHandlerText returns the http.Handler for /1337.
func (r *HTTPReceiver) DebuggerCountersHandlerText() http.Handler {
	return r.debuggerCounters.handlerText()
}

// DebuggerCountersHandlerJSON returns the http.Handler for /1337.json.
func (r *HTTPReceiver) DebuggerCountersHandlerJSON() http.Handler {
	return r.debuggerCounters.handlerJSON()
}
