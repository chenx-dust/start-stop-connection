package ssc

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type Process struct {
	Command  []string
	ExitChan chan struct{}

	proc   *exec.Cmd
	paused bool
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
		log.Println("process start error: ", err)
		return err
	}

	go func() {
		cmd.Wait()
		p.ExitChan <- struct{}{}
	}()

	p.proc = cmd
	p.paused = false
	return nil
}

func (p *Process) StartInteractive() error {
	cmd := exec.Command(p.Command[0], p.Command[1:]...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	go func() {
		defer ptmx.Close()
		defer term.Restore(int(os.Stdin.Fd()), oldState)
		go io.Copy(ptmx, os.Stdin)
		io.Copy(os.Stdout, ptmx)
		p.ExitChan <- struct{}{}
	}()

	p.proc = cmd
	p.paused = false
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
	p.paused = true
	return p.signal(syscall.SIGSTOP)
}

func (p *Process) Resume() error {
	p.paused = false
	return p.signal(syscall.SIGCONT)
}

func (p *Process) Stop() error {
	return p.signal(syscall.SIGTERM)
}

func (p *Process) Kill() error {
	return p.signal(syscall.SIGKILL)
}

func (p *Process) IsPaused() bool {
	return p.paused
}
