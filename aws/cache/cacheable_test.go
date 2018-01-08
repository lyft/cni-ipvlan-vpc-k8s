package cache

import (
	"encoding/json"
	"testing"
	"time"
)

type Foo struct {
	TheTime JSONTime `json:"time"`
}

type Thing struct {
	Value  string `json:"value"`
	TValue int
}

func TestJSONTime_MarshalJSON(t *testing.T) {
	input := Foo{JSONTime{time.Date(2017, 1, 1, 1, 1, 0, 0, time.UTC)}}
	output, err := json.Marshal(&input)
	if err != nil {
		t.Error(err)
	}
	if string(output) != `{"time":"2017-01-01T01:01:00Z"}` {
		t.Error(string(output))
	}
}

func TestJSONTime_UnmarshalJSON(t *testing.T) {
	input := []byte(`{"time":"2017-01-01T01:01:00Z"}`)
	var foo Foo
	err := json.Unmarshal(input, &foo)
	if err != nil {
		t.Error(err)
	}
	expected := Foo{JSONTime{time.Date(2017, 1, 1, 1, 1, 0, 0, time.UTC)}}
	if !foo.TheTime.Time.Equal(expected.TheTime.Time) {
		t.Errorf("Times were not equal: %v %v %v", foo.TheTime.Time, expected.TheTime.Time, foo)
	}
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
