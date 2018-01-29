package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nightlyone/lockfile"
)

// LockfileRun wraps execution of a specified function around a file lock
func LockfileRun(run func() error) error {
	lock, err := lockfile.New(filepath.Join(os.TempDir(), "cni-ipvlan-vpc-k8s.lock"))
	if err != nil {
		return err
	}
	tries := 1000

	for {
		tries--
		if tries <= 0 {
			return fmt.Errorf("Lockfile not acquired, aborting")
		}

		err = lock.TryLock()
		if err == nil {
			break
		} else if err == lockfile.ErrBusy {
			time.Sleep(100 * time.Millisecond)
		} else if err == lockfile.ErrNotExist {
			time.Sleep(100 * time.Millisecond)
		} else {
			return err
		}
	}

	defer lock.Unlock()
	return run()
}
