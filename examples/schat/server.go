package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/crewjam/tlshttp"
)

func serve(listener net.Listener) {
	certificate, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		panic(err)
	}

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %s", err)
		}

		tlsConn := tls.Server(c, &tls.Config{
			Certificates: []tls.Certificate{certificate},
		})
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("tls: %s", err)
		}

		go func() {
			io.Copy(os.Stdout, tlsConn)
			c.Close()
		}()

		go func() {
			io.Copy(tlsConn, os.Stdin)
			c.Close()
		}()
	}
}

func main() {
	var err error
	var listener net.Listener
	if true {
		listener, err = tlshttp.Listen()
		if err != nil {
			panic(err)
		}
		http.Handle("/", listener.(http.Handler))
		go http.ListenAndServe(":10000", nil)
	} else {
		listener, err = net.Listen("tcp", ":10000")
		if err != nil {
			panic(err)
		}
	}
	go serve(listener)

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt)
	_ = <-exitSignal
}
