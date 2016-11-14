package boot

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"rsc.io/letsencrypt"
)

// TLSConfig returns a instance of tls.Config configured with strict defaults.
func TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// Do not include RC4 or 3DES
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		NextProtos: []string{"http/1.1"},
	}
}

// LetsEncryptCertManager returns functions that can be set for tls.Config.GetCertificate
// that uses Let's Encrypt to auto-register and refresh certs.
func LetsEncryptCertManager(cache storage.Store, domains []string) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	var m letsencrypt.Manager
	m.SetHosts(domains)

	sort.Strings(domains)
	cacheFilename := strings.Join(domains, ",") + ".cert-cache"
	b, _, err := cache.Get(cacheFilename)
	if err != nil {
		if errors.Cause(err) != storage.ErrNoObject {
			golog.Errorf("Failed to load cert cache '%s': %s", cacheFilename, err)
		}
	} else {
		if err := m.Unmarshal(string(b)); err != nil {
			golog.Errorf("Failed to unmarshal cert cache: %s", err)
		}
	}

	go func() {
		for range m.Watch() {
			golog.Infof("Saving cert state")
			state := m.Marshal()
			if _, err := cache.Put(cacheFilename, []byte(state), "application/binary", nil); err != nil {
				golog.Errorf("Failed to write cert cache: %s", err)
			}
		}
	}()

	return m.GetCertificate
}

// HTTPSListenAndServe is a replacement for srv.ListenAndServe that
// includes optional proxy protocol support.
func HTTPSListenAndServe(srv *http.Server, proxyProtocol bool) error {
	conn, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return errors.Trace(err)
	}
	conn = tcpKeepAliveListener{conn.(*net.TCPListener)}
	if proxyProtocol {
		conn = &proxyproto.Listener{Listener: conn}
	}
	return srv.Serve(tls.NewListener(conn, srv.TLSConfig))
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away. (borrowed from net/http)
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, errors.Errorf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, nil
	}
}

// SelfSignedCertificate generates a certificate list with a single self signed certificate in it
func SelfSignedCertificate() ([]tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Trace(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Spruce Health Self Signed"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cert := &bytes.Buffer{}
	key := &bytes.Buffer{}
	pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	block, err := pemBlockForKey(priv)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pem.Encode(key, block)
	tlsCert, err := tls.X509KeyPair(cert.Bytes(), key.Bytes())
	if err != nil {
		return nil, errors.Trace(err)
	}
	return []tls.Certificate{tlsCert}, nil
}
