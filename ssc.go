package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chenx-dust/start-stop-connection/ssc"
)

func main() {
	var portMapping string
	var freezeDelay time.Duration
	var interactive bool

	flag.StringVar(&portMapping, "p", "", "port mapping")
	flag.DurationVar(&freezeDelay, "d", 0, "freeze delay")
	flag.BoolVar(&interactive, "i", false, "interactive")
	flag.Parse()

	if portMapping == "" {
		log.Panicln("port mapping is required")
	}
	if flag.NArg() == 0 {
		log.Panicln("process command is required")
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
		Command: flag.Args(),
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

	freezeTimer := time.NewTimer(freezeDelay)
	for {
		select {
		case val := <-connChan:
			if val {
				totalConn++
				log.Println("new connection")
				if totalConn == 1 {
					if err := proc.Resume(); err != nil {
						log.Panicln("process resume error: ", err)
					}
					freezeTimer.Stop()
					log.Println("resume process")
				}
			} else {
				totalConn--
				log.Println("connection closed")
				if totalConn == 0 {
					freezeTimer.Reset(freezeDelay)
				}
			}
		case <-freezeTimer.C:
			if totalConn == 0 {
				if err := proc.Pause(); err != nil {
					log.Println("process pause error: ", err)
				}
				log.Println("pause process")
			}
		}
	}
}
