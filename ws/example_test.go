package ws_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// A Server and a Client exchange frames. Both sides deliver every received
// frame to a callback; here the server echoes.
func Example() {
	ctx := context.Background()
	srv, err := ws.NewServer(ws.ServerConfig{
		Versions: []ocpp.Version{ocpp.V16},
		OnFrame: func(c *ws.Conn, frame []byte) {
			_ = c.Write(ctx, []byte("echo: "+string(frame)))
		},
	})
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go srv.Serve(ln)
	defer srv.Shutdown(ctx)

	connected := make(chan *ws.Conn, 1)
	received := make(chan string, 1)
	client, err := ws.NewClient(ws.ClientConfig{
		URL:       "ws://" + ln.Addr().String() + "/ocpp",
		ID:        "CS001",
		Versions:  []ocpp.Version{ocpp.V16},
		OnConnect: func(c *ws.Conn) { connected <- c },
		OnFrame:   func(_ *ws.Conn, frame []byte) { received <- string(frame) },
	})
	if err != nil {
		panic(err)
	}
	go client.Run(ctx)
	defer client.Close()

	conn := <-connected
	fmt.Println("connected as", conn.ID(), "using", conn.Version())
	_ = conn.Write(ctx, []byte("hello"))
	fmt.Println(<-received)
	// Output:
	// connected as CS001 using ocpp1.6
	// echo: hello
}

// An Authenticator decides whether a handshake may proceed. This one implements
// HTTP Basic authentication (security profiles 1 and 2).
func ExampleBasicAuthenticator() {
	auth := ws.BasicAuthenticator(func(id string) (string, bool) {
		return "correct-horse-battery", id == "CS001"
	})
	try := func(id, password string) error {
		r := httptest.NewRequest(http.MethodGet, "/ocpp/"+id, nil)
		r.SetBasicAuth(id, password)
		return auth(r, id)
	}
	fmt.Println(try("CS001", "correct-horse-battery"))
	fmt.Println(try("CS001", "guess"))
	fmt.Println(try("CS999", "anything"))
	// Output:
	// <nil>
	// ws: unauthorized
	// ws: unknown charge point
}

// ServerTLSConfig and ClientTLSConfig enforce TLS 1.2 or newer and the cipher
// suites OCPP 2.0.1 allows.
func ExampleCipherSuites() {
	fmt.Println(len(ws.CipherSuites()) > 0)
	// Output: true
}
