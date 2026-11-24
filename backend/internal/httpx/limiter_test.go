package httpx

import (
	"sync"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(perSecond float64, burst int) (*Limiter, *clock) {
	c := &clock{t: time.Unix(1_000_000, 0)}
	l := NewLimiter(perSecond, burst)
	l.now = c.now
	return l, c
}

func TestAllowSpendsBurstThenRefuses(t *testing.T) {
	l, _ := newTestLimiter(1, 3)
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d refused inside the burst", i+1)
		}
	}
	ok, wait := l.Allow("a")
	if ok {
		t.Fatal("request over the burst was allowed")
	}
	if wait <= 0 || wait > time.Second {
		t.Fatalf("wait = %v, want up to 1s", wait)
	}
}

func TestRefillOverTimeAndCap(t *testing.T) {
	l, c := newTestLimiter(2, 4)
	for i := 0; i < 4; i++ {
		l.Allow("a")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("bucket should be empty")
	}
	c.advance(500 * time.Millisecond)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("one token should have refilled after 500ms at 2/s")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("only one token should have refilled")
	}
	c.advance(time.Hour)
	for i := 0; i < 4; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d refused after a long idle time", i+1)
		}
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("refill must be capped at the burst")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l, _ := newTestLimiter(1, 1)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a refused")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("a allowed twice")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b must not share a's bucket")
	}
}

func TestCheckHitReset(t *testing.T) {
	l, _ := newTestLimiter(0.001, 2)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Check("k"); !ok {
			t.Fatal("Check must not spend tokens")
		}
	}
	l.Hit("k")
	l.Hit("k")
	l.Hit("k")
	if ok, wait := l.Check("k"); ok || wait <= 0 {
		t.Fatalf("after two hits Check = %v, %v, want refused with a wait", ok, wait)
	}
	l.Reset("k")
	if ok, _ := l.Check("k"); !ok {
		t.Fatal("Reset must give the key a full bucket")
	}
}

func TestLimiterPerWindowMatchesAttempts(t *testing.T) {
	l := NewLimiterPer(5, 15*time.Minute)
	c := &clock{t: time.Unix(2_000_000, 0)}
	l.now = c.now
	for i := 0; i < 5; i++ {
		l.Hit("k")
	}
	ok, wait := l.Check("k")
	if ok || wait < 2*time.Minute || wait > 3*time.Minute+time.Second {
		t.Fatalf("Check = %v, %v, want refused for about 3 minutes (15m / 5)", ok, wait)
	}
}

func TestIdleKeysAreSwept(t *testing.T) {
	l, c := newTestLimiter(1, 2)
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	if l.Len() != 3 {
		t.Fatalf("Len = %d, want 3", l.Len())
	}
	c.advance(2 * time.Minute)
	l.Allow("d")
	if l.Len() != 1 {
		t.Fatalf("Len = %d after the sweep, want only the fresh key", l.Len())
	}
}

func TestZeroRateGivesLongWait(t *testing.T) {
	l, _ := newTestLimiter(0, 1)
	l.Allow("a")
	if _, wait := l.Allow("a"); wait != time.Hour {
		t.Fatalf("wait = %v, want 1h when nothing refills", wait)
	}
}

func TestLimiterIsSafeForConcurrentUse(t *testing.T) {
	l := NewLimiter(1000, 100)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				l.Allow("shared")
				l.Check("shared")
				l.Hit("other")
			}
		}()
	}
	wg.Wait()
}
