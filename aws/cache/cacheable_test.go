package cache

import (
	"testing"
	"time"
)

type Thing struct {
	Value  string `json:"value"`
	TValue int
}

func TestGet(t *testing.T) {
	var d Thing
	state := Get("key_not_exist", &d)
	if state != CacheNoEntry {
		t.Errorf("Empty cache did not return a valid state %v", state)
	}

	d.Value = "Hello"
	d.TValue = 12

	state = Store("hello", 30*time.Second, &d)
	if state != CacheFound {
		t.Errorf("Invalid store of the cache key %v", state)
	}

	var e Thing
	state = Get("hello", &e)
	if state != CacheFound {
		t.Errorf("Can't reload existing key %v", state)
	}

	if d.Value != e.Value && d.TValue != e.TValue {
		t.Errorf("%v != %v", d, t)
	}

	// Test expiration
	Store("hello2", 1*time.Millisecond, &d)
	time.Sleep(100 * time.Millisecond)
	state = Get("hello2", &e)
	if state != CacheExpired {
		t.Error("Cache did not expire")
	}
}
