package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/crewjam/tlshttp"
)

func serve(listener net.Listener) {
	for {
		c, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %s", err)
		}

		go func() {
			io.Copy(os.Stdout, c)
			c.Close()
		}()

		go func() {
			io.Copy(c, os.Stdin)
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
