package ssc

import (
	"io"
	"log"
	"net"
	"sync"
)

type Forwarder struct {
	ListenAddr *net.TCPAddr
	DestAddr   *net.TCPAddr
	ConnChan   chan<- bool

	listener *net.TCPListener
}

func (f *Forwarder) Listen() error {
	listener, err := net.ListenTCP("tcp", f.ListenAddr)
	if err != nil {
		return err
	}
	f.listener = listener
	go f.listenLoop()
	return nil
}

func (f *Forwarder) listenLoop() {
	for {
		listenConn, err := f.listener.AcceptTCP()
		if err != nil {
			log.Println("listen error: ", err)
			continue
		}
		fwdConn, err := net.DialTCP("tcp", nil, f.DestAddr)
		if err != nil {
			log.Println("dial error: ", err)
			continue
		}
		go f.forwardLoop(listenConn, fwdConn)
	}
}

func (f *Forwarder) forwardLoop(listenConn, fwdConn *net.TCPConn) {
	var wg sync.WaitGroup
	wg.Add(2)
	f.ConnChan <- true
	forward := func(src, dst *net.TCPConn) {
		defer wg.Done()
		_, err := io.Copy(dst, src)
		if err != nil {
			log.Println("forward close due to: ", err)
			src.Close()
			dst.Close()
		}
	}
	go forward(listenConn, fwdConn)
	go forward(fwdConn, listenConn)
	wg.Wait()
	f.ConnChan <- false
}
