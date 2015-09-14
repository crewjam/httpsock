package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"

	"github.com/crewjam/tlshttp"
)

func main() {
	var err error
	var c net.Conn

	if true {
		u, _ := url.Parse("http://localhost:10000")
		c, err = tlshttp.Dial(u)
	} else {
		c, err = net.Dial("tcp", "localhost:10000")
	}
	if err != nil {
		panic(err)
	}
	defer c.Close()

	tlsConn := tls.Client(c, &tls.Config{InsecureSkipVerify: true})
	if err := tlsConn.Handshake(); err != nil {
		panic(err)
	}

	exitSignal := make(chan os.Signal, 1)

	go func() {
		io.Copy(tlsConn, os.Stdin)
		c.Close()
		exitSignal <- os.Interrupt
	}()
	go func() {
		io.Copy(os.Stdout, tlsConn)
		c.Close()
		exitSignal <- os.Interrupt
	}()

	signal.Notify(exitSignal, os.Interrupt)
	_ = <-exitSignal
}
