package lib

import (
	"encoding/json"
	"time"
)

// JSONTime is a RFC3339 encoded time with JSON marshallers
type JSONTime struct {
	time.Time
}

// MarshalJSON marshals a JSONTime to an RFC3339 string
func (j *JSONTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Time.Format(time.RFC3339))
}

// UnmarshalJSON unmarshals a JSONTime to a time.Time
func (j *JSONTime) UnmarshalJSON(js []byte) error {
	var rawString string
	err := json.Unmarshal(js, &rawString)
	if err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339, rawString)
	if err != nil {
		return err
	}
	j.Time = t
	return nil
}
