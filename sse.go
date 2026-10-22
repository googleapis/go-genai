// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"iter"
	"log"
	"math"
	"strconv"
	"time"
)

// The scanner grows from initialSSEBufferSize as needed and fails with bufio.ErrTooLong past
// maxSSELineSize. Both override the scanner's 64KB default.
const (
	initialSSEBufferSize = 1024
	maxSSELineSize       = 268435456 // 256MB
)

const defaultSSEEventType = "message"

// sseEvent is one event dispatched by an event stream.
type sseEvent struct {
	eventType   string
	data        []byte
	lastEventID string
	retry       time.Duration
}

// sseBuffers is the parser state the SSE specification keeps while reading a stream. The data and
// event type buffers reset on every dispatch; the last event ID and reconnection time persist.
type sseBuffers struct {
	data        []byte
	hasData     bool
	eventType   string
	lastEventID string
	retry       time.Duration
}

// dispatch returns the event that a blank line or the end of the stream dispatches and resets the
// buffers that do not persist. It returns nil when no data field arrived since the last dispatch.
func (b *sseBuffers) dispatch() *sseEvent {
	if !b.hasData {
		b.eventType = ""
		return nil
	}
	event := &sseEvent{
		eventType:   b.eventType,
		data:        b.data,
		lastEventID: b.lastEventID,
		retry:       b.retry,
	}
	if event.eventType == "" {
		event.eventType = defaultSSEEventType
	}
	b.data, b.hasData, b.eventType = nil, false, ""
	return event
}

// iterateSSEStream parses r as an event stream and yields its events in order, per
// https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation
//
// The specification says to ignore an undefined field. This parser reports one instead, as a
// stream can carry a server error in place of SSE fields and dropping it would leave the caller
// with a truncated response. Iteration continues, so events sharing the stream are still yielded.
func iterateSSEStream(r io.Reader) iter.Seq2[*sseEvent, error] {
	return func(yield func(*sseEvent, error) bool) {
		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, initialSSEBufferSize), maxSSELineSize)
		s.Split(scanSSELines)

		var b sseBuffers
		firstLine := true

		for s.Scan() {
			line := s.Bytes()
			if firstLine {
				// A byte order mark at the start of the stream is not part of a field name.
				line = bytes.TrimPrefix(line, []byte("\ufeff"))
				firstLine = false
			}

			// A blank line dispatches the event.
			if len(line) == 0 {
				if event := b.dispatch(); event != nil && !yield(event, nil) {
					return
				}
				continue
			}
			// A leading colon marks a comment, such as the ": heartbeat" keep-alives that
			// prevent an idle connection from timing out.
			if line[0] == ':' {
				continue
			}
			name, value, found := bytes.Cut(line, []byte(":"))
			if !found {
				// A line with no colon is a field name whose value is empty.
				name, value = line, nil
			} else {
				// A single leading space after the colon is part of the delimiter.
				value = bytes.TrimPrefix(value, []byte(" "))
			}
			switch string(name) {
			case "event":
				b.eventType = string(value)
			case "data":
				if b.hasData {
					b.data = append(b.data, '\n')
				}
				// value points into the scanner's buffer, which the next Scan reuses.
				b.data = append(b.data, value...)
				b.hasData = true
			case "id":
				// A value containing NUL cannot be sent back in a Last-Event-ID header, so
				// the spec ignores the field rather than the character.
				if !bytes.ContainsRune(value, 0) {
					b.lastEventID = string(value)
				}
			case "retry":
				// ParseUint rejects a sign and any digit grouping, leaving exactly the ASCII
				// digits the spec allows. A count too large for a Duration is ignored.
				if ms, err := strconv.ParseUint(string(value), 10, 64); err == nil && ms <= math.MaxInt64/uint64(time.Millisecond) {
					b.retry = time.Duration(ms) * time.Millisecond
				}
			default:
				if !yield(nil, invalidChunkError(line)) {
					return
				}
			}
		}
		// A stream may end without a final blank line.
		if event := b.dispatch(); event != nil && !yield(event, nil) {
			return
		}
		if err := s.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				log.Printf("The response is too large to process in streaming mode. Please use a non-streaming method.")
			}
			log.Printf("Error %v", err)
			yield(nil, err)
		}
	}
}

// scanSSELines splits an event stream into lines terminated by CRLF, LF, or a lone CR, per the SSE
// specification. bufio.ScanLines is not a substitute: it does not recognize a lone CR, and it
// discards a trailing empty line, which in SSE dispatches an event.
func scanSSELines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			return i + 1, data[0:i:i], nil
		}
		// A trailing CR is ambiguous: the next read may deliver the LF that pairs with it.
		// Splitting now would leave that LF behind as an extra empty line.
		if i == len(data)-1 && !atEOF {
			return 0, nil, nil
		}
		if i+1 < len(data) && data[i+1] == '\n' {
			return i + 2, data[0:i:i], nil // CRLF
		}
		return i + 1, data[0:i:i], nil // lone CR
	}
	// A final line that the stream did not terminate.
	if atEOF {
		return len(data), data[0:len(data):len(data)], nil
	}
	return 0, nil, nil
}
