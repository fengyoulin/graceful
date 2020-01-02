package graceful

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	envKey         string
	envFdsKey      string
	isGraceful     bool
	inheritedFiles []*os.File
)

func init() {
	base := strings.ToUpper(filepath.Base(os.Args[0]))
	envKey = base + "_GRACEFUL"
	envFdsKey = base + "_GRACEFUL_FDS"
	if os.Getenv(envKey) == "true" {
		isGraceful = true
	}
	if cntStr := os.Getenv(envFdsKey); cntStr != "" {
		cnt, err := strconv.ParseInt(cntStr, 10, 64)
		if err != nil {
			log.Fatalf("invalid fds in env: %s", cntStr)
		}
		inheritedFiles = make([]*os.File, cnt)
		for i := 0; i < int(cnt); i++ {
			inheritedFiles[i] = os.NewFile(uintptr(3+i), "")
		}
	}
}

// Start a new process with extra files
func Start(files []*os.File) (cmd *exec.Cmd, err error) {
	oe := os.Environ()
	cnt := len(files)
	env := make([]string, 0, len(oe)+cnt)
	for _, v := range oe {
		if !strings.HasPrefix(v, envKey) && !strings.HasPrefix(v, envFdsKey) {
			env = append(env, v)
		}
	}
	if cnt > 0 {
		env = append(env, envKey+"=true")
		env = append(env, envFdsKey+"="+strconv.FormatInt(int64(cnt), 10))
	}

	cmd = exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.ExtraFiles = files

	err = cmd.Start()
	return
}

// IsGraceful just as its name
func IsGraceful() bool {
	return isGraceful
}

// GetInheritedFiles just as its name
func GetInheritedFiles() []*os.File {
	return inheritedFiles
}
