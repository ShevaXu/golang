package sys

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// CmdHook gives access to the executed Cmd
// and its stdin.
type CmdHook func(*exec.Cmd, io.WriteCloser)

// SignalExit sends SIGNAL to exit the process.
func SignalExit(sig syscall.Signal) CmdHook {
	return func(cmd *exec.Cmd, stdin io.WriteCloser) {
		// ignore err here
		cmd.Process.Signal(sig)
		return
	}
}

// RunCmd runs a command with specific arg in a blocking way;
// it calls doneFunc when the context is done.
func RunCmd(ctx context.Context, name, arg, wd, logFile string, afterStartFunc, doneFunc CmdHook) (err error) {
	// look for binary path
	path, err := exec.LookPath(name)
	if err != nil {
		return
	}

	// convert arg string to args slices
	args := strings.Fields(arg)
	cmd := exec.Command(path, args...)

	// specify working directory
	if wd != "" {
		cmd.Dir = wd
	}

	// open log file
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		cmd.Stdout = f
		cmd.Stderr = f
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	if err = cmd.Start(); err != nil {
		return
	}

	if afterStartFunc != nil {
		afterStartFunc(cmd, stdin)
	}

	cancel := make(chan struct{})
	done := ctx.Done()

	// exit handling
	go func() {
		select {
		case <-done:
			if doneFunc != nil {
				// if done function provided
				doneFunc(cmd, stdin)
			} else {
				SignalExit(syscall.SIGTERM)(cmd, stdin)
			}
		case <-cancel:
			return
		}
	}()

	err = cmd.Wait()

	// cleanup the exit handling goroutine
	close(cancel)

	return err
}

// GetFileMd5 computes the file MD5 as IO stream;
// note that this might takes time for large file.
func GetFileMd5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CheckPid checks if the process of pid exists.
// Ref: https://stackoverflow.com/questions/15204162/check-if-a-process-exists-in-go-way
func CheckPid(pid int) error {
	// On Unix systems, FindProcess always succeeds and
	// returns a Process for the given pid, regardless of whether the process exists.
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// For Unix system:
	// If sig is 0, then no signal is sent, but error checking is still
	// performed; this can be used to check for the existence of a process ID
	// or process group ID.
	return p.Signal(syscall.Signal(0))
}
