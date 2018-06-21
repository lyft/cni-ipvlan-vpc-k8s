package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/lyft/cni-ipvlan-vpc-k8s/lib"
)

const (
	registryDir           = "cni-ipvlan-vpc-k8s"
	registryFile          = "registry.json"
	registrySchemaVersion = 1
)

func defaultRegistry() registryContents {
	return registryContents{
		SchemaVersion: registrySchemaVersion,
		IPs:           map[string]*registryIP{},
	}
}

type registryIP struct {
	ReleasedOn lib.JSONTime `json:"released_on"`
}

type registryContents struct {
	SchemaVersion int                    `json:"schema_version"`
	IPs           map[string]*registryIP `json:"ips"`
}

// Registry defines a re-usable IP registry which tracks IPs that are
// free in the system and when they were last released back to the pool.
type Registry struct {
	path string
	lock sync.Mutex
}

// registryPath gives a default location for the registry
// which varies based on invoking user ID
func registryPath() string {
	uid := os.Getuid()
	if uid != 0 {
		// Non-root users of the registry
		return path.Join("/run/user", fmt.Sprintf("%d", uid), registryDir)
	}

	return path.Join("/run", registryDir)
}

func (r *Registry) ensurePath() (string, error) {
	if len(r.path) == 0 {
		r.path = registryPath()
	}
	rpath := r.path
	info, err := os.Stat(rpath)
	if err != nil || !info.IsDir() {
		err = os.MkdirAll(rpath, os.ModeDir|0700)
		if err != nil {
			return "", err
		}
	}
	rpath = path.Join(rpath, registryFile)
	return rpath, nil
}

func (r *Registry) load() (*registryContents, error) {
	// Load the pre-versioned schema
	contents := defaultRegistry()
	rpath, err := r.ensurePath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(rpath)
	if os.IsNotExist(err) {
		// Return an empty registry, prefilled with IPs
		// already existing on all interfaces and timestamped
		// at the golang epoch
		free, err := FindFreeIPsAtIndex(0, false)
		if err == nil {
			for _, freeAlloc := range free {
				contents.IPs[freeAlloc.IP.String()] = &registryIP{lib.JSONTime{time.Time{}}}
			}
			err = r.save(&contents)
			return &contents, err
		}
		return &contents, nil
	} else if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	if decoder == nil {
		return nil, fmt.Errorf("invalid decoder")
	}

	err = decoder.Decode(&contents)
	if err != nil {
		log.Printf("invalid registry format, returning empty registry %v", err)
		contents = defaultRegistry()
	}

	// Reset the registry if the version is not what we can deal with.
	// Add more states here, or just let the registry be blown away
	// on invalid loads
	if contents.SchemaVersion != registrySchemaVersion {
		contents = defaultRegistry()
	}
	if contents.IPs == nil {
		contents = defaultRegistry()
	}
	return &contents, nil
}

func (r *Registry) save(rc *registryContents) error {
	rpath, err := r.ensurePath()
	if err != nil {
		return err
	}
	file, err := os.OpenFile(rpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if encoder == nil {
		return fmt.Errorf("could not make a new encoder")
	}
	rc.SchemaVersion = registrySchemaVersion
	err = encoder.Encode(rc)
	return err
}

// TrackIP records an IP in the free registry with the current system
// time as the current freed-time. If an IP is freed again, the time
// will be updated to the new current time.
func (r *Registry) TrackIP(ip net.IP) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	contents, err := r.load()
	if err != nil {
		return err
	}

	contents.IPs[ip.String()] = &registryIP{lib.JSONTime{time.Now()}}
	return r.save(contents)
}

// ForgetIP removes an IP from the registry
func (r *Registry) ForgetIP(ip net.IP) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	contents, err := r.load()
	if err != nil {
		return err
	}

	delete(contents.IPs, ip.String())

	return r.save(contents)
}

// HasIP checks if an IP is in an registry
func (r *Registry) HasIP(ip net.IP) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	contents, err := r.load()
	if err != nil {
		return false, err
	}

	_, ok := contents.IPs[ip.String()]
	return ok, nil
}

// TrackedBefore returns a list of all IPs last recorded time _before_
// the time passed to this function. You probably want to call this
// with time.Now().Add(-duration).
func (r *Registry) TrackedBefore(t time.Time) ([]net.IP, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	contents, err := r.load()
	if err != nil {
		return nil, err
	}

	returned := []net.IP{}
	for ipString, entry := range contents.IPs {
		if entry.ReleasedOn.Before(t) {
			ip := net.ParseIP(ipString)
			if ip == nil {
				continue
			}
			returned = append(returned, ip)
		}
	}
	return returned, nil
}

// Clear clears the registry unconditionally
func (r *Registry) Clear() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	rpath, err := r.ensurePath()
	if err != nil {
		return err
	}

	err = os.Remove(rpath)
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

// List returns a list of all tracked IPs
func (r *Registry) List() (ret []net.IP, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	contents, err := r.load()
	if err != nil {
		return nil, err
	}
	for stringIP := range contents.IPs {
		ret = append(ret, net.ParseIP(stringIP))
	}
	return
}

// Jitter takes a duration and adjusts it forward by a up to `pct` percent
// uniformly.
func Jitter(d time.Duration, pct float64) time.Duration {
	jitter := rand.Int63n(int64(float64(d) * pct))
	d += time.Duration(jitter)
	return d
}
