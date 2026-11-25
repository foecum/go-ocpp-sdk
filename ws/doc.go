// Package ws is the WebSocket transport for OCPP-J, for both roles. It does
// framing of nothing itself: it moves text frames between a peer and your code,
// and plugs into package ocppj.
//
// # Server (CSMS)
//
// A Server is an http.Handler. It reads the charge point identity from the last
// path segment of the URL, authenticates the handshake, negotiates the OCPP
// version from the Sec-WebSocket-Protocol header, and delivers each received
// frame to OnFrame:
//
//	srv, _ := ws.NewServer(ws.ServerConfig{
//		Versions:      []ocpp.Version{ocpp.V201, ocpp.V16},
//		Authenticator: ws.BasicAuthenticator(lookupPassword),
//		OnFrame:       func(c *ws.Conn, frame []byte) { /* feed an ocppj.Endpoint */ },
//	})
//	http.ListenAndServe(":8080", srv) // or srv.ListenAndServe(":8080")
//
// A charge point that connects again under the same identity replaces its older
// connection. Shutdown closes every connection gracefully (close code 1001).
//
// # Security profiles
//
//	Profile 1: BasicAuthenticator over ws://
//	Profile 2: BasicAuthenticator, and ServerTLSConfig(cert, nil) over wss://
//	Profile 3: ServerTLSConfig(cert, clientCAs) with ClientCertAuthenticator
//
// TLS 1.2 is the minimum and only the cipher suites listed by CipherSuites are
// enabled, as OCPP 2.0.1 requires.
//
// # Keep-alive
//
// Both sides send WebSocket pings (every 30 seconds by default) and close a
// connection whose peer stops answering. A closed connection reports why through
// Conn.Err, for example ErrPingTimeout or ErrReplaced.
package ws
