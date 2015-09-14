# httpsock

`httpsock` is a (hairbrained?) effort to implement a socket that uses HTTP requests as a transport. The socket implementation should be good enough that TLS could be tunneled over HTTP.

## HTTPS?

HTTPS is HTTP over TLS. This is TLS over HTTP. The other way around.

## Are you kidding!?

At this point, I'm not sure. Maybe. Maybe not.

A serious use of this might be to replace websockets and/or HTTP/2 streams from behind proxies that don't support them.

It may be nessesary to provide additional security on top of an existing secure channel for networks that implement SSL man-in-the-middle.

# Protocol

The client initiates a session my making an HTTP POST request to a URL

    POST /api/v1/socket HTTP/1.1
    Host: api.example.com

The server responds with a session ID that must be provided for future requests:

    HTTP/1.1 201 Created
    X-Session: 7acb54ebbb72368c7d968e5273c458e7ebfc772cd710afca77320170a3f83ddd
    Content-Length: 0

To send data to the server, make a PUT request:

    PUT /api/v1/socket HTTP/1.1
    Host: api.example.com
    X-Session: 7acb54ebbb72368c7d968e5273c458e7ebfc772cd710afca77320170a3f83ddd
    Content-Length: 13

    Hello, World!

To receive data from the server, perform a GET request with a `X-Content-Length` header to specify the maximum number of bytes that you will accept.

    GET /api/v1/socket HTTP/1.1
    Host: api.example.com
    X-Session: 7acb54ebbb72368c7d968e5273c458e7ebfc772cd710afca77320170a3f83ddd
    X-Content-Length: 32768

The server will respond with your data:

    HTTP/1.1 200 OK
    Content-Length: 13

    Hello, World!

To close a connection perform a DELETE:

    DELETE /api/v1/socket HTTP/1.1
    Host: api.example.com
    X-Session: 7acb54ebbb72368c7d968e5273c458e7ebfc772cd710afca77320170a3f83ddd

The server will respond with `204 No Content`.

# Example

A client:

        u, _ := url.Parse("http://localhost:10000")
        c, err = tlshttp.Dial(u)
        f err != nil {
            panic(err)
        }
        defer c.Close()

        tlsConn := tls.Client(c, &tls.Config{InsecureSkipVerify: true})
        if err := tlsConn.Handshake(); err != nil {
            panic(err)
        }
        fmt.Fprintf(tlsConn, "Hello, World!")

A server:

        listener, err := tlshttp.Listen()
        if err != nil {
           panic(err)
        }
        http.Handle("/", listener.(http.Handler))
        go http.ListenAndServe(":10000", nil)

        certificate, err := tls.LoadX509KeyPair("server.crt", "server.key")
        if err != nil {
            panic(err)
        }

        for {
            c, err := listener.Accept()
            if err != nil {
                panic(err)
            }

            tlsConn := tls.Server(c, &tls.Config{
                Certificates: []tls.Certificate{certificate},
            })
            if err := tlsConn.Handshake(); err != nil {
                panic(err)
            }

            greeting, err := bufio.NewReader(tlsConn).ReadString('\n')
            if err != nil {
                panic(err)
            }
            fmt.Printf("got %q\n", greeting)
            c.Close()
        }

# TODO

- Write tests
- Get my head around all the various states of sockets
- Implement timeouts.
- Handle long timeouts.  
- Make sure I have my head around the various error states (EOF, closed, etc.) and that they are faithfully brought across the http channel
- Less round trips
- `X-Session` as a key is certainly a wrong thing to do. Think about the right (reasonably secure) way to differentiate the connections
- Cache busting
