package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/go-proxy-protocol/proxyproto"
)

// The local cert and key are only used when the Debug config
// option is set and TLSCert and TLSKey config options are
// not defined. The check to make sure that this holds is in
// utils.*Config.Validate().
var (
	localTLSCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDGDCCAgKgAwIBAgIRAOvlgNu24IVI52mjWfaHiQIwCwYJKoZIhvcNAQELMBIx
EDAOBgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAw
MDBaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQCiMm/EAvYlu+eRDdcBqxcGKO59vrxVkSz8QLVShajUPl4jWFo8xZHG
MsNBLmUXFulkIRQStvFzfpo9/QHWDyvUmrNMy5P/LE54x9EO/kmjJu1B8ReRqdyD
WsEej3RM9WBo+fISY+2yMMHbN/3PuZzIHVMGl45/PcXuCs7OMYOQWgn0yURYSvP/
ltwDrLxebgLV13S3fk9iJf9CjBV6beEMjAPbm6I+s4mtJff/74ci7nHkMyxOT1PS
w5HJW6fdmFpiId5tJd9k4MNmkRPnxHlKxwCjGi0JzAKA32qYgqqPb8OBg2RP+5JD
usC+2e/ohx2s/TZO+lPb+LsUyKDjmAFtAgMBAAGjazBpMA4GA1UdDwEB/wQEAwIA
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MDEGA1UdEQQq
MCiCDiouc3BydWNlLmxvY2FshwR/AAABhxAAAAAAAAAAAAAAAAAAAAABMAsGCSqG
SIb3DQEBCwOCAQEAST9NUS/YQKpj9oFY6QOR4tDro+UTlN5DkMVUBacX/alDj58q
bPFs6XwsPWnbA3ZQtWq0zMaOyFWcj1jH5tsc5RUVDbhcUmrhwc1MdzWYfiTMgLMp
7M59n0dt3icL6WYWeM+Gb2YB1wIe9I2MxqB7RZnMPocyaEjXA06wfWVrsYOH+0XV
UQ95EPwON7Izclw7CVQHnYXYK3uGtFjIRO7d9EC0KZjcURb+rKvzY0+74CxyyW/9
MInfGcQScPfqGGmUoBw8tA7wRJYLCbCTAIa3F83ikPsm/T/xg6kuml1IwNfyPiMY
b56u79PVFpIezbwxtGnZodROrIrSXHUs3OAlIg==
-----END CERTIFICATE-----`)
	localTLSKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAojJvxAL2JbvnkQ3XAasXBijufb68VZEs/EC1UoWo1D5eI1ha
PMWRxjLDQS5lFxbpZCEUErbxc36aPf0B1g8r1JqzTMuT/yxOeMfRDv5JoybtQfEX
kancg1rBHo90TPVgaPnyEmPtsjDB2zf9z7mcyB1TBpeOfz3F7grOzjGDkFoJ9MlE
WErz/5bcA6y8Xm4C1dd0t35PYiX/QowVem3hDIwD25uiPrOJrSX3/++HIu5x5DMs
Tk9T0sORyVun3ZhaYiHebSXfZODDZpET58R5SscAoxotCcwCgN9qmIKqj2/DgYNk
T/uSQ7rAvtnv6IcdrP02TvpT2/i7FMig45gBbQIDAQABAoIBAQCIwpRApuqbWHvh
b9T5gCQyunKVLi0ozPcsXvdEdJStGUVQ8h9sHH5Uqtq96/uq41O5bLa7LOwboQU2
/Uz+C96+Lg6+0uyf/ODRsFHTHZBDdAAbWMixtpLLYstxFC5Q8ZjwCsgUv5NdawUZ
7XUiIHRUu30VEtdA7Homw5Aqhc9T92+rlWASJdMD9WQJ+h1xQDcnqb/LsihnV6Od
01rT4DOtDfcgJsgHzseCUOiJiuQ5c/AILiVWB+atNxRbsSHV175/nllIbX/C9UOF
WuuAvXPhhRoFX4CxVEhseNQQKpqlX0FK6dibYC+aWkiclaqvd/52LX9CXYVYs34s
C6VarDkVAoGBANEr9tJhFG+H5VaSMLp55uhlmTJ6JwBne9sF0HC8MHgKIg4G8v2H
UDRQ98oi8hzgRKz3xvrd2wLCEaPAQSfx5cY7tGiR6Y/fwyX/uRakD2a8dMPhhttq
2Vt4x0QrFahZRLoMF1NiOcaNwHrzRm6YP7vm9X2CjYdWj+CBqWGIF7drAoGBAMaC
Qr+vwhr/9Wsmnmh5OK7lE5IWV8tjh+fnLjU5FflNykKOs0nhNDQFw59XATcKEti3
+FSvK9DYOSNU38li+njzHb2mnQlYjae616IcyudWW6J3LRCereUHJxjyO9szEbwK
VNER9ncg7LoJIBa0YATkI3Jc95jJUk6RBQdt/NiHAoGAOwBBsPn9P7B/ejnmUNNN
1MPDwL8//RczkoZDU2lh6ppBHN/M7sKaVwd3vaa50HdaJ8gEcoLd4htHyn7SYigT
fiUdMFnoHdMqQq+tT7ubNIl4DkCxP3cWNH0PCCV3CHOVtTzv329XiLA3WPcCKPP9
Fk2BdZO7xC8gil1In+A5gF0CgYBISsn6OwzKfmqnEhJgY70j3GMLMb3ZYS7uYn+u
fFKnTxAYuxVKE4zKYUsDrVDQ9Yc1i5IRbRXc4dG1L0Ssd7JV99vd5F6ON8Smz+GV
tTyjkQygFxy/T7pujPNNH3Jy+p87xttqpEsIyWHMwmQAQMIzJc5O6NJ2vuKNoDyf
nwuU4wKBgQC9ByjK1nGl5xbgQmWBn+smBZhWwY46lDLIxWFXpHzTbAFlVJlUp7H4
HjuhdXVa8R908jY2phFN9NjkEFXyUUuar9VClE5V5NSD8WYsAkPrrpYNw3BQVJRP
J15BERU7hluvxXOZn5wenPP0DcDqZX/34dNPE58CKtzlDP/UlpSqzQ==
-----END RSA PRIVATE KEY-----`)
)

func serve(conf *Config, hand http.Handler) {
	server := &http.Server{
		Addr:    conf.ListenAddr,
		Handler: hand,
		// FIXME: 5 minute timeout is to allow for media uploads/downloads
		// These long running requests should be handled separately instead of requiring
		// the entire API to have such long timeouts.
		ReadTimeout:    5 * time.Minute,
		WriteTimeout:   5 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

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

		tlsServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Forwarded-Proto", "https")
			hand.ServeHTTP(w, r)
		})

		var certs tls.Certificate

		if conf.TLSCert != "" && conf.TLSKey != "" {
			cert, err := conf.ReadURI(conf.TLSCert)
			if err != nil {
				log.Fatal(err)
			}
			key, err := conf.ReadURI(conf.TLSKey)
			if err != nil {
				log.Fatal(err)
			}
			certs, err = tls.X509KeyPair(cert, key)
			if err != nil {
				log.Fatal(err)
			}
		} else if conf.Debug {
			golog.Warningf("Using local TLS keys")
			var err error
			certs, err = tls.X509KeyPair(localTLSCert, localTLSKey)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("No TLS keys provided and Debug not true")
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

	golog.Infof("Starting server on %s...", conf.ListenAddr)
	log.Fatal(server.ListenAndServe())
}
