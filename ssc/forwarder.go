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
			log.Println("listen error:", err)
			continue
		}
		log.Println("new connection:", listenConn.RemoteAddr())
		go func() {
			f.ConnChan <- true
			defer func() { f.ConnChan <- false }()
			fwdConn, err := net.DialTCP("tcp", nil, f.DestAddr)
			if err != nil {
				log.Println("dial error:", err)
				return
			}
			tx, rx := f.forwardLoop(listenConn, fwdConn)
			log.Println("connection closed:", listenConn.RemoteAddr(), "tx:", tx, "rx:", rx, "total:", tx+rx)
		}()
	}
}

func (f *Forwarder) forwardLoop(listenConn, fwdConn *net.TCPConn) (tx, rx int64) {
	var wg sync.WaitGroup
	wg.Add(2)
	forward := func(src, dst *net.TCPConn, n_ *int64) {
		defer wg.Done()
		n, err := io.Copy(dst, src)
		if err != nil {
			log.Println("forward close due to:", err)
		}
		*n_ = n
		src.Close()
		dst.Close()
	}
	go forward(listenConn, fwdConn, &tx)
	go forward(fwdConn, listenConn, &rx)
	wg.Wait()
	return
}
