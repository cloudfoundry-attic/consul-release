package confab

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type AgentRunner struct {
	Path      string
	PIDFile   string
	ConfigDir string
	Stdout    io.Writer
	Stderr    io.Writer
	Recursors []string
	cmd       *exec.Cmd
}

func isRunningProcess(pidFilePath string) bool {
	pidFileContents, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(pidFileContents))
	if err != nil {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func (r *AgentRunner) Run() error {
	if isRunningProcess(r.PIDFile) {
		return fmt.Errorf("consul_agent is already running, please stop it first")
	}

	if _, err := os.Stat(r.ConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("Config dir does not exist: %s", r.ConfigDir)
	}

	args := []string{
		"agent",
		fmt.Sprintf("-config-dir=%s", r.ConfigDir),
	}

	for _, recursor := range r.Recursors {
		args = append(args, fmt.Sprintf("-recursor=%s", recursor))
	}

	r.cmd = exec.Command(r.Path, args...)
	r.cmd.Stdout = r.Stdout
	r.cmd.Stderr = r.Stderr

	err := r.cmd.Start()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(r.PIDFile, []byte(fmt.Sprintf("%d", r.cmd.Process.Pid)), 0644)
	if err != nil {
		return fmt.Errorf("error writing PID file: %s", err)
	}

	go r.cmd.Wait() // reap child process if it dies

	return nil
}

func (r *AgentRunner) getProcess() (*os.Process, error) {
	pidFileContents, err := ioutil.ReadFile(r.PIDFile)
	if err != nil {
		return nil, err
	}

	pid, err := strconv.Atoi(string(pidFileContents))
	if err != nil {
		return nil, err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err // not tested. As of Go 1.5, FindProcess never errors
	}

	return process, nil
}

func (r *AgentRunner) Wait() error {
	process, err := r.getProcess()
	if err != nil {
		return err
	}

	for {
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (r *AgentRunner) Stop() error {
	process, err := r.getProcess()
	if err != nil {
		return err
	}

	err = process.Signal(syscall.Signal(syscall.SIGKILL))
	if err != nil {
		return err
	}

	return nil
}
