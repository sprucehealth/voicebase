package main

import (
	"carefront/libs/golog"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/SpruceHealth/go-proxy-protocol/proxyproto"
)

func serve(conf *Config, hand http.Handler) {
	server := &http.Server{
		Addr:           conf.ListenAddr,
		Handler:        hand,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if conf.TLSCert != "" && conf.TLSKey != "" {
		go func() {
			// Make a copy of the server to avoid sharing internal state
			// (currently there is none but it's safer not to assume that)
			tlsServer := *server
			tlsServer.TLSConfig = &tls.Config{
				MinVersion:               tls.VersionTLS10,
				PreferServerCipherSuites: true,
				CipherSuites: []uint16{
					// Do not include RC4 or 3DES
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
			}
			if tlsServer.TLSConfig.NextProtos == nil {
				tlsServer.TLSConfig.NextProtos = []string{"http/1.1"}
			}

			cert, err := conf.ReadURI(conf.TLSCert)
			if err != nil {
				log.Fatal(err)
			}
			key, err := conf.ReadURI(conf.TLSKey)
			if err != nil {
				log.Fatal(err)
			}
			certs, err := tls.X509KeyPair(cert, key)
			if err != nil {
				log.Fatal(err)
			}

			tlsServer.TLSConfig.Certificates = []tls.Certificate{certs}

			conn, err := net.Listen("tcp", conf.TLSListenAddr)
			if err != nil {
				log.Fatal(err)
			}

			if conf.ProxyProtocol {
				conn = &proxyproto.Listener{Listener: conn}
			}

			ln := tls.NewListener(conn, tlsServer.TLSConfig)

			golog.Infof("Starting SSL server on %s...", conf.TLSListenAddr)
			log.Fatal(tlsServer.Serve(ln))
		}()
	}
	golog.Infof("Starting server on %s...", conf.ListenAddr)

	log.Fatal(server.ListenAndServe())
}
