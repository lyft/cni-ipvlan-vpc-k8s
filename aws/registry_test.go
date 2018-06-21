package aws

import (
	"net"
	"testing"
	"time"
)

const (
	IP1 = "127.0.0.1"
	IP2 = "127.0.0.2"
	IP3 = "127.0.0.3"
)

func TestRegistry_TrackIp(t *testing.T) {
	r := &Registry{}

	err := r.TrackIP(net.ParseIP(IP1))
	if err != nil {
		t.Fatalf("Failed to track IP %v", err)
	}

	if ok, err := r.HasIP(net.ParseIP(IP1)); !ok || err != nil {
		t.Fatalf("Did not remember IP %v %v %v", IP1, ok, err)
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := &Registry{}

	err := r.Clear()
	if err != nil {
		t.Fatalf("clear failed %v", err)
	}

	r.TrackIP(net.ParseIP(IP3))
	if ok, err := r.HasIP(net.ParseIP(IP3)); !ok || err != nil {
		t.Fatalf("Did not remember IP %v %v %v", IP3, ok, err)
	}

	err = r.Clear()
	if err != nil {
		t.Fatalf("clear failed %v", err)
	}

	if ok, err := r.HasIP(net.ParseIP(IP3)); ok || err != nil {
		t.Fatalf("Did not forget IP %v %v %v", IP3, ok, err)
	}
}

func TestRegistry_ForgetIP(t *testing.T) {
	r := &Registry{}

	err := r.Clear()
	if err != nil {
		t.Fatalf("clear failed %v", err)
	}

	r.TrackIP(net.ParseIP(IP2))
	if ok, err := r.HasIP(net.ParseIP(IP2)); !ok || err != nil {
		t.Fatalf("Did not remember IP %v %v %v", IP3, ok, err)
	}

	err = r.ForgetIP(net.ParseIP(IP2))
	if err != nil {
		t.Fatalf("forget failed %v", err)
	}

	if ok, err := r.HasIP(net.ParseIP(IP2)); ok || err != nil {
		t.Fatalf("Did not forget IP %v %v %v", IP2, ok, err)
	}

	// Forget an IP never registered
	err = r.ForgetIP(net.ParseIP(IP1))
	if err != nil {
		t.Fatalf("forgetting an IP not tracked should not be an error")
	}

}

func TestRegistry_TrackedBefore(t *testing.T) {
	r := &Registry{}

	err := r.Clear()
	if err != nil {
		t.Fatalf("clear failed %v", err)
	}

	r.TrackIP(net.ParseIP(IP1))
	now := time.Now()

	before, err := r.TrackedBefore(now.Add(100 * time.Hour))
	if err != nil {
		t.Fatalf("error tracked before %v", err)
	}

	if len(before) != 1 {
		t.Fatalf("Invalid number of entries, got %v", before)
	}

	after, err := r.TrackedBefore(now.Add(-100 * time.Hour))
	if err != nil {
		t.Fatalf("error tracked before %v", err)
	}

	if len(after) != 0 {
		t.Fatalf("Should return no IPs before the future, got %v", after)
	}
}

func TestJitter(t *testing.T) {
	d1 := 1 * time.Second
	d1p := Jitter(d1, 0.10)
	if d1 >= d1p {
		t.Fatalf("Jitter did not move forward: %v %v", d1, d1p)
	}

	if d1p > 1101*time.Millisecond {
		t.Fatalf("Jitter moved more than 10pct forward %v", d1p)
	}
}
