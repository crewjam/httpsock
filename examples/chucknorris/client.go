package main

import (
	"bufio"
	"fmt"
	"net/url"

	"github.com/crewjam/tlshttp"
)

func main() {
	u, _ := url.Parse("http://localhost:10000")
	c, err := tlshttp.Dial(u)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	//name := "Chuck Norris"
	//fmt.Fprintf(c, "%s\n", name)
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		fmt.Printf("%s\n", scanner.Text())
		break
	}
}
