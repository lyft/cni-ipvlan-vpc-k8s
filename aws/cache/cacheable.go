package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
)

const (
	cacheRoot    = "/var/run/user"
	cacheProgram = "cni-ipvlan-vpc-k8s"
)

// State defines the return of the Store and Get calls
type State int

const (
	// CacheFound means the key was found and valid
	CacheFound State = iota
	// CacheExpired means the key was found, but has expired. The value returned is not valid.
	CacheExpired
	// CacheNoEntry means the key was not found.
	CacheNoEntry
	// CacheNotAvailable means the cache system is not working as expected and has an internal error
	CacheNotAvailable
)

func cachePath() string {
	return path.Join(cacheRoot, fmt.Sprintf("%d", os.Getuid()), cacheProgram)
}

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

// Cacheable defines metadata for objects which can be cached to files as JSON
type Cacheable struct {
	Expires  JSONTime    `json:"_expires"`
	Contents interface{} `json":contents"`
}

func ensureDirectory() error {
	cachePath := cachePath()
	info, err := os.Stat(cachePath)
	if err == nil && info.IsDir() {
		return nil
	}

	err = os.Mkdir(cachePath, os.ModeDir|0700)
	return err
}

// Get gets a key from the named cache file
func Get(key string, decodeTo interface{}) State {
	err := ensureDirectory()
	if err != nil {
		return CacheNotAvailable
	}

	file, err := os.Open(path.Join(cachePath(), key))
	if err != nil {
		return CacheNoEntry
	}

	defer file.Close()

	var contents Cacheable
	contents.Contents = decodeTo
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&contents)
	if err != nil {
		return CacheNoEntry
	}

	if contents.Expires.Time.Before(time.Now()) {
		return CacheExpired
	}

	return CacheFound
}

// Store stores the given data interface as a JSON file with a given expiration time
// under the given key.
func Store(key string, lifetime time.Duration, data interface{}) State {
	err := ensureDirectory()
	if err != nil {
		return CacheNotAvailable
	}

	key = path.Join(cachePath(), key)

	file, err := os.OpenFile(key, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return CacheNotAvailable
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if encoder == nil {
		return CacheNotAvailable
	}

	var contents Cacheable
	contents.Expires.Time = time.Now().Add(lifetime)
	contents.Contents = data
	err = encoder.Encode(&contents)
	if err != nil {
		return CacheNotAvailable
	}

	return CacheFound
}
