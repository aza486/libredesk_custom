package email

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/knadh/smtppool"
)

// Exercise actual SMTP transport: an explicit plaintext relay must work, while
// STARTTLS/TLS settings must never silently fall back to plaintext.
func TestSMTPConfiguredTransport(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12}
	for _, tc := range []struct {
		name, mode                   string
		implicit, startTLS, skipCert bool
		wantError                    string
	}{
		{name: "explicit plaintext", mode: "none"},
		{name: "STARTTLS required", mode: "starttls", wantError: "STARTTLS extension not found"},
		{name: "default requires STARTTLS", wantError: "STARTTLS extension not found"},
		{name: "STARTTLS supported", mode: "starttls", startTLS: true, skipCert: true},
		{name: "implicit TLS", mode: "tls", implicit: true, skipCert: true},
		{name: "certificate verification retained", mode: "tls", implicit: true, wantError: "certificate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { listener.Close() })
			if tc.implicit {
				listener = tls.NewListener(listener, tlsConfig)
			}
			delivered := make(chan string, 1)
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				p := textproto.NewConn(conn)
				p.PrintfLine("220 localhost test SMTP")
				for {
					line, err := p.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO") && tc.startTLS:
						p.PrintfLine("250-localhost\r\n250 STARTTLS")
					case line == "STARTTLS":
						p.PrintfLine("220 Ready for TLS")
						secure := tls.Server(conn, tlsConfig)
						if secure.Handshake() != nil {
							return
						}
						p = textproto.NewConn(secure)
					case line == "DATA":
						p.PrintfLine("354 Send message")
						body, err := p.ReadDotBytes()
						if err != nil {
							return
						}
						delivered <- string(body)
						p.PrintfLine("250 Accepted")
					case line == "QUIT":
						p.PrintfLine("221 Bye")
						return
					default:
						p.PrintfLine("250 OK")
					}
				}
			}()
			addr := listener.Addr().(*net.TCPAddr)
			pools, err := NewSmtpPool([]imodels.SMTPConfig{{Host: "127.0.0.1", Port: addr.Port, TLSType: tc.mode, TLSSkipVerify: tc.skipCert, AuthProtocol: "none", MaxConns: 1, MaxMessageRetries: 1, PoolWaitTimeout: "2s"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			pool := pools[0]
			err = pool.Send(smtppool.Email{From: "sender@example.test", To: []string{"recipient@example.test"}, Subject: "transport regression", Text: []byte("transport regression body")})
			pool.Close()
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q, got %v", tc.wantError, err)
				}
				select {
				case <-delivered:
					t.Fatal("message sent despite failed TLS requirement")
				default:
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				select {
				case body := <-delivered:
					if !strings.Contains(body, "transport regression body") {
						t.Fatal("SMTP body lost")
					}
				case <-time.After(time.Second):
					t.Fatal("SMTP message not received")
				}
			}
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				t.Fatal("SMTP test server did not stop")
			}
		})
	}
}
