package tlshttp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type errNetwork struct {
	Status     string
	StatusCode int
}

func (err *errNetwork) Error() string {
	return fmt.Sprintf("%d %s", err.StatusCode, err.Status)
}

// Dial establishes a new connection
func Dial(addr *url.URL) (net.Conn, error) {
	resp, err := http.Post(addr.String(), "application/x-socket", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, &errNetwork{Status: resp.Status, StatusCode: resp.StatusCode}
	}
	sessionID := resp.Header.Get("X-Session")
	return &Conn{addr: addr, sessionID: sessionID}, nil
}

// Conn represents an active connection
type Conn struct {
	addr      *url.URL
	sessionID string
}

func (c *Conn) Read(b []byte) (n int, err error) {
	req, err := http.NewRequest("GET", c.addr.String(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-Session", c.sessionID)
	req.Header.Set("X-Content-Length", fmt.Sprintf("%d", len(b)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, &errNetwork{Status: resp.Status, StatusCode: resp.StatusCode}
	}

	nbytes, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return 0, err
	}
	n, err = resp.Body.Read(b[:nbytes])
	if n > 0 && err == io.EOF {
		err = nil
	}
	return n, err
}

func (c *Conn) Write(b []byte) (n int, err error) {
	req, err := http.NewRequest("PUT", c.addr.String(), bytes.NewBuffer(b))
	if err != nil {
		return -1, err
	}
	req.Header.Set("X-Session", c.sessionID)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return -1, &errNetwork{Status: resp.Status, StatusCode: resp.StatusCode}
	}
	return len(b), nil
}

func (c *Conn) Close() error {
	req, err := http.NewRequest("DELETE", c.addr.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Session", c.sessionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return &errNetwork{Status: resp.Status, StatusCode: resp.StatusCode}
	}
	return nil
}

type fakeLocalAddr struct{}

func (f fakeLocalAddr) Network() string {
	return "http"
}
func (f fakeLocalAddr) String() string {
	return "http"
}

func (c *Conn) LocalAddr() net.Addr {
	return fakeLocalAddr{}
}

func randomSessionID() []byte {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != 32 {
		panic(err)
	}
	return buf
}

type remoteAddr struct {
	str string
}

func (f remoteAddr) Network() string {
	return "http"
}
func (f remoteAddr) String() string {
	return f.str
}

func (c *Conn) RemoteAddr() net.Addr {
	return remoteAddr{str: c.addr.String()}
}

func (c *Conn) SetDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}

func Listen() (net.Listener, error) {
	l := Listener{
		Sessions:    map[string]*Session{},
		newSessions: make(chan *Session),
	}
	return &l, nil
}

type Listener struct {
	Sessions    map[string]*Session
	newSessions chan *Session
}

type Session struct {
	ID         []byte
	ClientConn net.Conn
	ServerConn net.Conn
}

func (s *Session) Read(b []byte) (n int, err error) {
	return s.ClientConn.Read(b)
}

func (s *Session) Write(b []byte) (n int, err error) {
	return s.ClientConn.Write(b)
}

func (s *Session) Close() error {
	s.ClientConn.Close()
	s.ServerConn.Close()
	// TODO(ross): remove from the upstream listener array (w/o creating a cycle)
	return nil
}

func (s *Session) LocalAddr() net.Addr {
	return fakeLocalAddr{} // ???
}

func (s *Session) RemoteAddr() net.Addr {
	return remoteAddr{str: fmt.Sprintf("%x", s.ID)}
}

func (s *Session) SetDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}
func (s *Session) SetReadDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}
func (s *Session) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
	return nil
}

func httpBadRequest(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

const maxReadBytes = 16 * 1024 * 1024

func (l *Listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.Sessions == nil {
		l.Sessions = map[string]*Session{}
	}
	if r.Method == "POST" {
		s := &Session{ID: randomSessionID()}
		s.ClientConn, s.ServerConn = net.Pipe()

		l.Sessions[string(s.ID)] = s
		l.newSessions <- s

		w.Header().Add("X-Session", fmt.Sprintf("%x", s.ID))
		log.Printf("SESSION %x", s.ID)
		w.WriteHeader(http.StatusCreated)
		return
	}

	sessionIDString := r.Header.Get("X-Session")
	if sessionIDString == "" {
		httpBadRequest(w)
		return
	}
	sessionID, err := hex.DecodeString(sessionIDString)
	if err != nil {
		httpBadRequest(w)
		return
	}
	session, ok := l.Sessions[string(sessionID)]
	if !ok {
		httpBadRequest(w)
		return
	}

	if r.Method == "PUT" {
		io.Copy(session.ServerConn, r.Body)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == "GET" {
		nbytes, err := strconv.ParseInt(r.Header.Get("X-Content-Length"), 10, 64)
		if err != nil || nbytes <= 0 || nbytes > maxReadBytes {
			httpBadRequest(w)
			return
		}
		buf := make([]byte, nbytes)
		n, err := session.ServerConn.Read(buf)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", n))
		w.WriteHeader(http.StatusOK)
		w.Write(buf[:n])
		return
	}

	if r.Method == "DELETE" {
		session.Close()
		delete(l.Sessions, string(session.ID))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	httpBadRequest(w)
}

func (l *Listener) Accept() (net.Conn, error) {
	for session := range l.newSessions {
		return session, nil
	}
	return nil, fmt.Errorf("listener is closed")
}

func (l *Listener) Close() error {
	close(l.newSessions)
	return nil
}

func (l *Listener) Addr() net.Addr {
	return fakeLocalAddr{}
}
