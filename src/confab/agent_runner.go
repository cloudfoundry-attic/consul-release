package confab

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/pivotal-golang/lager"
)

type AgentRunner struct {
	Path      string
	PIDFile   string
	ConfigDir string
	Stdout    io.Writer
	Stderr    io.Writer
	Recursors []string
	Logger    logger
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
		err := fmt.Errorf("consul_agent is already running, please stop it first")
		r.Logger.Error("agent-runner.run.consul-already-running", err)
		return err
	}

	if _, err := os.Stat(r.ConfigDir); os.IsNotExist(err) {
		err := fmt.Errorf("config dir does not exist: %s", r.ConfigDir)
		r.Logger.Error("agent-runner.run.config-dir-missing", err)
		return err
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

	r.Logger.Info("agent-runner.run.start", lager.Data{
		"cmd":  r.Path,
		"args": args,
	})
	err := r.cmd.Start()
	if err != nil {
		r.Logger.Error("agent-runner.run.start.failed", errors.New(err.Error()), lager.Data{
			"cmd":  r.Path,
			"args": args,
		})
		return err
	}

	go r.cmd.Wait() // reap child process if it dies

	r.Logger.Info("agent-runner.run.success")
	return nil
}

func (r *AgentRunner) WritePID() error {
	r.Logger.Info("agent-runner.run.write-pidfile", lager.Data{
		"pid":  r.cmd.Process.Pid,
		"path": r.PIDFile,
	})

	if err := ioutil.WriteFile(r.PIDFile, []byte(fmt.Sprintf("%d", r.cmd.Process.Pid)), 0644); err != nil {
		err = fmt.Errorf("error writing PID file: %s", err)
		r.Logger.Error("agent-runner.run.write-pidfile.failed", err, lager.Data{
			"pid":  r.cmd.Process.Pid,
			"path": r.PIDFile,
		})
		return err
	}

	return nil
}

func (r *AgentRunner) getProcess() (*os.Process, error) {
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process, nil
	}

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
	r.Logger.Info("agent-runner.wait.get-process")

	process, err := r.getProcess()
	if err != nil {
		r.Logger.Error("agent-runner.wait.get-process.failed", errors.New(err.Error()))
		return err
	}

	r.Logger.Info("agent-runner.wait.get-process.result", lager.Data{
		"pid": process.Pid,
	})

	r.Logger.Info("agent-runner.wait.signal", lager.Data{
		"pid": process.Pid,
	})

	for {
		err = process.Signal(syscall.Signal(0))
		if err == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		break
	}

	r.Logger.Info("agent-runner.wait.success")
	return nil
}

func (r *AgentRunner) Stop() error {
	r.Logger.Info("agent-runner.stop.get-process")

	process, err := r.getProcess()
	if err != nil {
		r.Logger.Error("agent-runner.stop.get-process.failed", errors.New(err.Error()))
		return err
	}

	r.Logger.Info("agent-runner.stop.get-process.result", lager.Data{
		"pid": process.Pid,
	})

	r.Logger.Info("agent-runner.stop.signal", lager.Data{
		"pid": process.Pid,
	})

	err = process.Signal(syscall.Signal(syscall.SIGKILL))
	if err != nil {
		r.Logger.Error("agent-runner.stop.signal.failed", err)
		return err
	}

	r.Logger.Info("agent-runner.stop.success")
	return nil
}

func (r *AgentRunner) Cleanup() error {
	r.Logger.Info("agent-runner.cleanup.remove", lager.Data{
		"pidfile": r.PIDFile,
	})

	if err := os.Remove(r.PIDFile); err != nil {
		err = errors.New(err.Error())
		r.Logger.Error("agent-runner.cleanup.remove.failed", err, lager.Data{
			"pidfile": r.PIDFile,
		})
		return err
	}

	r.Logger.Info("agent-runner.cleanup.success")

	return nil
}
