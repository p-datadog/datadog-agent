// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package eventbuf

import (
	"sync/atomic"

	"github.com/DataDog/datadog-agent/pkg/dyninst/output"
)

// SharedMessage refcounts a single underlying Message so it can be attached
// to multiple Readys. NotePanicUnwoundRange uses this to fan one synthetic
// recovery event out across N user-probe finalizations.
//
// Lifecycle:
//   - Construct with NewSharedMessage(underlying).
//   - For each fanout target, call Acquire() to get a refcounting handle
//     that satisfies the Message interface.
//   - Pass each handle into a buffer call (e.g. attached as a return-side
//     fragment).
//   - As the consumer drains each Ready, the handle's Release runs and
//     decrements. When the last handle Releases, the underlying Message's
//     Release is called once.
//
// After the last Acquire call has happened, the caller signals "no more
// Acquires" by invoking ReleaseBase(). If any handles were taken, the
// last handle's Release() will release the underlying; if none were
// taken, ReleaseBase() releases it immediately. Not safe to call Acquire
// after ReleaseBase().
type SharedMessage struct {
	underlying Message
	refs       atomic.Int32
}

// NewSharedMessage wraps underlying so it can be shared across many handles.
func NewSharedMessage(underlying Message) *SharedMessage {
	return &SharedMessage{underlying: underlying}
}

// Acquire returns a handle holding one reference on the underlying message.
// Each handle is its own Message; calling Release on it decrements the
// shared refcount.
func (s *SharedMessage) Acquire() Message {
	s.refs.Add(1)
	return sharedMessageHandle{shared: s}
}

// ReleaseBase signals the end of the Acquire phase. If no handles were
// taken (refs == 0), the underlying message is released now. If handles
// were taken, this is a no-op: the last handle's Release() decrements
// refs to zero and releases the underlying. Must be called exactly once,
// after all Acquires have happened, and before any new Acquires (none
// are valid after this returns).
func (s *SharedMessage) ReleaseBase() {
	if s.refs.Load() == 0 {
		s.underlying.Release()
	}
}

func (s *SharedMessage) release() {
	if s.refs.Add(-1) != 0 {
		return
	}
	s.underlying.Release()
}

// sharedMessageHandle is one reference into a SharedMessage. Each handle
// implements Message in its own right; the buffer doesn't need to know it's
// a shared payload.
type sharedMessageHandle struct {
	shared *SharedMessage
}

// Event returns the shared underlying event bytes. The buffer reads this
// to determine fragment length; all handles return the same slice.
func (h sharedMessageHandle) Event() output.Event {
	return h.shared.underlying.Event()
}

// Release decrements the shared refcount; the underlying message is
// released when the last handle releases.
func (h sharedMessageHandle) Release() {
	h.shared.release()
}
