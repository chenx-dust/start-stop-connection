package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chenx-dust/start-stop-connection/ssc"
)

func main() {
	var portMapping string
	var freezeDelay time.Duration
	var interactive bool
	var napDuration time.Duration

	flag.StringVar(&portMapping, "p", "", "port mapping (format: listenAddr=destAddr;..., example: :8080=:80;127.0.0.1:8081=127.0.0.1:81)")
	flag.DurationVar(&freezeDelay, "d", 0, "freeze delay (default: 0, format: go-style duration string)")
	flag.BoolVar(&interactive, "i", false, "interactive mode (using pty)")
	flag.DurationVar(&napDuration, "n", 0, "nap duration (quick pause when no long connection, default: 0)")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Println("process command is required")
		flag.Usage()
		os.Exit(1)
	}

	totalConn := 0
	connChan := make(chan bool)
	for _, mapping := range strings.Split(portMapping, ";") {
		if mapping == "" {
			continue
		}
		addrs := strings.Split(mapping, "=")
		if len(addrs) != 2 {
			log.Panicln("invalid port mapping with wrong format")
		}

		listenAddr, err := net.ResolveTCPAddr("tcp", addrs[0])
		if err != nil {
			log.Panicln("invalid listen address, error: ", err)
		}
		destAddr, err := net.ResolveTCPAddr("tcp", addrs[1])
		if err != nil {
			log.Panicln("invalid destination address, error: ", err)
		}

		forwarder := ssc.Forwarder{
			ListenAddr: listenAddr,
			DestAddr:   destAddr,
			ConnChan:   connChan,
		}
		if err := forwarder.Listen(); err != nil {
			log.Panicln("listen error: ", err)
		}
	}

	proc := ssc.Process{
		Command:  flag.Args(),
		ExitChan: make(chan struct{}),
	}
	log.SetOutput(os.Stdout)
	if interactive {
		if err := proc.StartInteractive(); err != nil {
			log.Panicln("process start error: ", err)
		}
	} else {
		if err := proc.Start(); err != nil {
			log.Panicln("process start error: ", err)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	freezeTimer := time.NewTimer(freezeDelay)
	isTiming := true
	lastFirstConnTime := time.Now()
	for {
		select {
		case val := <-connChan:
			if val {
				totalConn++
				if totalConn == 1 {
					log.Println("first connection")
					lastFirstConnTime = time.Now()
					if proc.IsPaused() {
						if err := proc.Resume(); err != nil {
							log.Panicln("process resume error: ", err)
						}
						log.Println("resume process")
					}
				}
			} else {
				totalConn--
				if totalConn == 0 {
					log.Println("last connection")
					if time.Since(lastFirstConnTime) < napDuration {
						// In napping mode, too short connection, continue to pause
						if !isTiming {
							freezeTimer.Reset(0)
							log.Println("nap mode, too short connection, continue to pause")
						}
					} else {
						isTiming = true
						freezeTimer.Reset(freezeDelay)
						log.Println("counting down to freeze")
					}
				}
			}
		case <-freezeTimer.C:
			isTiming = false
			if totalConn == 0 && !proc.IsPaused() {
				if err := proc.Pause(); err != nil {
					log.Println("process pause error: ", err)
				}
				log.Println("pause process, get into napping mode")
			}
		case <-proc.ExitChan:
			log.Println("process stopped")
			os.Exit(0)
		case signal := <-signalChan:
			log.Println("got signal: ", signal)
			if err := proc.Signal(signal.(syscall.Signal)); err != nil {
				log.Println("process signal error: ", err)
			}
			os.Exit(0)
		}
	}
}
