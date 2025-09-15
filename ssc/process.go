package ssc

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	Command []string

	proc *exec.Cmd
}

func (p *Process) Start() error {
	cmd := exec.Command(p.Command[0], p.Command[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Println("process start error: {}")
		return err
	}
	p.proc = cmd

	return nil
}

func (p *Process) signal(signum syscall.Signal) error {
	if p.proc == nil {
		return nil
	}

	if err := syscall.Kill(-p.proc.Process.Pid, signum); err != nil {
		log.Println("process kill error: {}")
		return err
	}

	return nil
}

func (p *Process) Pause() error {
	return p.signal(syscall.SIGSTOP)
}

func (p *Process) Resume() error {
	return p.signal(syscall.SIGCONT)
}

func (p *Process) Stop() error {
	return p.signal(syscall.SIGTERM)
}
