package lib

import (
	"encoding/json"
	"testing"
	"time"
)

type Foo struct {
	TheTime JSONTime `json:"time"`
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
