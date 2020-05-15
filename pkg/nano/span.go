package nano

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// MaxSpan is a range from the minimum possible time to the max possible time.
var MaxSpan = Span{Ts: 0, Dur: math.MaxInt64}

// Span represents a time span.  Spans are half-open: [Ts, Ts + Dur).
type Span struct {
	Ts
	Dur int64
}

type jsonSpan struct {
	Ts  Ts `json:"ts"`
	Dur Ts `json:"dur"`
}

// MarshalJSON fulfills the json.Marshaler interface.  We need to encode
// Dur as a 64-bit timestamp so we use jsonSpan to do the conversion.
func (s Span) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return json.Marshal(nil)
	}
	v := jsonSpan{s.Ts, Ts(s.Dur)}
	return json.Marshal(&v)
}

// UnmarshalJSON fulfills the json.Marshaler interface.  We need to encode
// Dur as a 64-bit timestamp so we use jsonSpan to do the conversion.
// XXX It would be nice to have Dur data type that we could marshal/unmarshal
// but that creates a lot of casting changes in the current code.  Maybe later.
func (s *Span) UnmarshalJSON(in []byte) error {
	if bytes.Equal(in, []byte("null")) {
		*s = Span{}
		return nil
	}
	var v jsonSpan
	if err := json.Unmarshal(in, &v); err != nil {
		return err
	}
	s.Ts = v.Ts
	s.Dur = int64(v.Dur)
	return nil
}

// NewSpanTs creates a Span from a Ts pair.  The Span is
// half-open: [start, end).
func NewSpanTs(start, end Ts) Span {
	return Span{Ts: start, Dur: int64(end - start)}
}

// End returns the first Ts after the Span (in other words, the smallest Ts
// greater than every Ts in the Span).
func (s Span) End() Ts {
	return s.Ts.Add(s.Dur)
}

// SubSpan divides the span into n subspans of approximately equal length and
// returns the i-th.
func (s Span) SubSpan(i, n int) Span {
	partitionSize := s.Dur / int64(n)
	start := s.Ts.Add(int64(i) * partitionSize)

	// Extend the final subspan to the end of s.  It is short by s.Dur % n
	// if integer division has truncated the value of partitionSize.
	if i == n-1 {
		partitionSize = int64(s.End() - start)
	}

	return Span{Ts: start, Dur: partitionSize}
}

// Partition divides the span into n subspans of approximately equal length and
// returns the index of the subspan containing ts.
func (s Span) Partition(ts Ts, n int) int {
	off := ts - s.Ts
	partitionSize := s.Dur / int64(n)
	i := int(int64(off) / partitionSize)

	// Fix the index if greater than that of the final span.  This happens
	// when ts > partitionSize * n, which in turn can happen when integer
	// division has truncated the value of partitionSize.
	if i > n-1 {
		panic("this shouldn't happen now")
		i = n - 1
	}

	return i
}

// MinDur returns the smallest duration >= minDur among spans
// that would be partioned in a span tree of degree fanout.
func (s Span) MinDur(minDur int64, fanout int) int64 {
	span := s
	for {
		child := span.SubSpan(0, fanout)
		if child.Dur < minDur {
			return span.Dur
		}
		span = child
	}
}

func MinDurForDay(minDur int64, fanout int) int64 {
	s := Span{Ts: 0, Dur: Day}
	return s.MinDur(minDur, fanout)
}

// Intersect merges two spans returning a new span representing the
// intesection of the two spans.  If the spans do not overlap, a zero valued
// span is returned.
func (s Span) Intersect(b Span) Span {
	start := max(s.Ts, b.Ts)
	end := min(s.End(), b.End())
	if start > end {
		return Span{}
	}
	return NewSpanTs(start, end)
}

// Union merges two spans returning a new span where start equals
// min(a.start, b.start) and end equals max(a.end, b.end). Assumes the two spans
// overlap.
func (s Span) Union(b Span) Span {
	return NewSpanTs(min(s.Ts, b.Ts), max(s.End(), b.End()))
}

// Subtract returns a slice of spans that represent the receiver span minus
// the time ranges of the input span. Assumes the two spans overlap.
func (s Span) Subtract(b Span) []Span {
	spans := []Span{}
	intersect := s.Intersect(b)
	if intersect.Ts > s.Ts {
		spans = append(spans, NewSpanTs(s.Ts, intersect.Ts))
	}
	if intersect.End() < s.End() {
		spans = append(spans, NewSpanTs(intersect.End(), s.End()))
	}
	return spans
}

// OverlapsOrAdjacent returns true if the two spans overlaps each or are adjacent.
func (s Span) OverlapsOrAdjacent(comp Span) bool {
	if s.Ts < comp.Ts {
		return comp.Ts <= s.End()
	} else {
		return s.Ts <= comp.End()
	}
}

// Overlaps returns true if the two spans overlap.
func (s Span) Overlaps(comp Span) bool {
	if s.Ts < comp.Ts {
		return comp.Ts < s.End()
	} else {
		return s.Ts < comp.End()
	}
}

// Contains returns true if the timestamp is in the time interval.
func (s Span) Contains(ts Ts) bool {
	return ts >= s.Ts && ts < s.End()
}

// ContainsClose returns true if the timestamp is in the time interval including
// the end of interval.
func (s Span) ContainsClosed(ts Ts) bool {
	return ts >= s.Ts && ts <= s.End()
}

// Covers returns true if the passed span is covered by s.
func (s Span) Covers(covered Span) bool {
	return s.Ts <= covered.Ts && s.End() >= covered.End()
}

func (s Span) String() string {
	d := time.Duration(s.Dur)
	return fmt.Sprintf("%s+%s", s.Ts.String(), d.String())
}

func (s Span) Pretty() string {
	d := time.Duration(s.Dur)
	return fmt.Sprintf("%s+%s", s.Ts.Pretty(), d.String())
}

func (s Span) IsZero() bool {
	return s.Dur == 0
}

func min(a, b Ts) Ts {
	if a < b {
		return a
	}
	return b
}

func max(a, b Ts) Ts {
	if a < b {
		return b
	}
	return a
}
