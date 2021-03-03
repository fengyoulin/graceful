package graceful

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
)

const (
	envKeySuffix    = "_GRACEFUL"
	envFdsKeySuffix = "_GRACEFUL_FDS"
)

var (
	initiated      int64
	envKey         string
	envFdsKey      string
	isGraceful     bool
	inheritedFiles []*os.File
	workerProcess  *os.Process
)

// Init ...
func Init(logger Logger) error {
	if !atomic.CompareAndSwapInt64(&initiated, 0, 1) {
		return nil
	}
	base := strings.ToUpper(filepath.Base(os.Args[0]))
	envKey = base + envKeySuffix
	envFdsKey = base + envFdsKeySuffix
	if os.Getenv(envKey) == "true" {
		isGraceful = true
	}
	if cntStr := os.Getenv(envFdsKey); cntStr != "" {
		cnt, err := strconv.ParseInt(cntStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid environment variable: %s=%s", envFdsKey, cntStr)
		}
		inheritedFiles = make([]*os.File, cnt)
		for i := 0; i < int(cnt); i++ {
			inheritedFiles[i] = os.NewFile(uintptr(3+i), "")
		}
	}
	if lg = logger; lg != nil {
		return nil
	}
	lg = defaultLogger()
	return nil
}

func startAndWait(files []*os.File) error {
	env := os.Environ()
	cnt := len(files)
	slc := make([]string, 0, len(env)+2)
	for _, v := range env {
		if !strings.HasPrefix(v, envKey) && !strings.HasPrefix(v, envFdsKey) {
			slc = append(slc, v)
		}
	}
	if cnt > 0 {
		slc = append(slc, envKey+"=true")
		slc = append(slc, envFdsKey+"="+strconv.FormatInt(int64(cnt), 10))
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = slc
	cmd.ExtraFiles = files

	err := cmd.Start()
	if err != nil {
		return err
	}

	workerProcess = cmd.Process

	if err = cmd.Wait(); err != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signal() == sigWorkerQuit {
			return nil
		}
	}
	return err
}
