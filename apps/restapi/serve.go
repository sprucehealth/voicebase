package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/go-proxy-protocol/proxyproto"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

// The local cert and key are only used when the Debug config
// option is set and TLSCert and TLSKey config options are
// not defined. The check to make sure that this holds is in
// utils.*Config.Validate().
var (
	localTLSCert = []byte(`-----BEGIN CERTIFICATE-----
MIIESDCCAzCgAwIBAgIJAPOdrq09k5OtMA0GCSqGSIb3DQEBBQUAMHUxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNp
c2NvMQ8wDQYDVQQKEwZTcHJ1Y2UxETAPBgNVBAsTCFNlY3VyaXR5MRUwEwYDVQQD
FAwqLnNwcnVjZS5sb2MwHhcNMTUwODA0MDEyNDIzWhcNMjUwODAxMDEyNDIzWjB1
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2Fu
IEZyYW5jaXNjbzEPMA0GA1UEChMGU3BydWNlMREwDwYDVQQLEwhTZWN1cml0eTEV
MBMGA1UEAxQMKi5zcHJ1Y2UubG9jMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAsJrWM2sDEWT8WruwbnR5uEBDBQEyNjBuMqYeq1fAKY7u0fkniLuJB8BH
oxyjGnKyieORDAXwJfDbd6RhE1FMQMUzfy2ziq9WglzFOK7MpLjnsa9+pfIO3Him
ivc0h2UUYws9m10F3UoxoNrgFcrIQnJJRn28P8NmLjGWBRJV71zqccfDNaxEjp+N
siLaRMVg2nvc4kjfAzWDCThYWKXwGNumZlYAXQr/ikgZdERUaI9cd8YVDZBuK3F5
CwDQJw38V10CzpMXaE7BENgc/G49gqEsAOtmraP+ryFpQIRv6+aeUfEDNnRJJwRA
k69fxGKj8sQNvmHfB7fq5IPuEovZpQIDAQABo4HaMIHXMB0GA1UdDgQWBBQGWOsT
/pZlmpG1Teqm0WrhrU2sLjCBpwYDVR0jBIGfMIGcgBQGWOsT/pZlmpG1Teqm0Wrh
rU2sLqF5pHcwdTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAU
BgNVBAcTDVNhbiBGcmFuY2lzY28xDzANBgNVBAoTBlNwcnVjZTERMA8GA1UECxMI
U2VjdXJpdHkxFTATBgNVBAMUDCouc3BydWNlLmxvY4IJAPOdrq09k5OtMAwGA1Ud
EwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBABwLxGYW8mulYYuZ6889SI+EaoTh
InQr2auy1nmOTKHdnu8u52MgANILBHPQ/D0/l0ZcO3Ta1/FLgddFHwKMS5n9m1TK
K6/LTmZ5ICDa7k3Kn4cJf6RZi1Y8Ip1FsOO3G7+Vq14uPG10mvWNDw/rIFy2cHPb
2gnaKmbMos2imqHfJqzPO/XbNA1+TeffGGEyGK1FtjGWJ6QEl7ttkbRbWhQyNdAk
DDFE/FB9pGeCsiwlbS+soIwtksBpN8LomYdO5+Fb5ZSlCFsbnKmuzl+X9QAGnRk5
ux/wdrQBJqZxDR5yTMtzRK7mBv15dS/15WqhQ/VYEycQ/FuRIkapSPpOzAg=
-----END CERTIFICATE-----`)
	localTLSKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsJrWM2sDEWT8WruwbnR5uEBDBQEyNjBuMqYeq1fAKY7u0fkn
iLuJB8BHoxyjGnKyieORDAXwJfDbd6RhE1FMQMUzfy2ziq9WglzFOK7MpLjnsa9+
pfIO3Himivc0h2UUYws9m10F3UoxoNrgFcrIQnJJRn28P8NmLjGWBRJV71zqccfD
NaxEjp+NsiLaRMVg2nvc4kjfAzWDCThYWKXwGNumZlYAXQr/ikgZdERUaI9cd8YV
DZBuK3F5CwDQJw38V10CzpMXaE7BENgc/G49gqEsAOtmraP+ryFpQIRv6+aeUfED
NnRJJwRAk69fxGKj8sQNvmHfB7fq5IPuEovZpQIDAQABAoIBAFAi0lOemVPJSSE6
zYIxZBIRRtf8hPZF35sn+f6x4MZ6zy+EKUZIIpPb0iXXlsMnjJA5LNYR07jDpDKT
6mDDoSA623U4HaIukcixo+RmnQGZzbi483UFc2zjal7gcXuGiEuxDBF1knWpdv+Q
keIRk/FivpE3+LXOSo1nfrVqboggFf5SEPhFMlqti/GuH/TH2W1aUjfSnmMBZXip
S/ND6TYuk9jQr9v5E3Nvhl7UCxCktlBbG5/2h4g10elluEgdElWdgfaqp2IyOHxG
qb4/drgD7HZW2Q2RryN+X9R2Nup7EOG6zM1vx/78uL103KIuEeaIPXBRprLtOv1Y
vzO1IMECgYEA4plX8LQKc0Ph6I1rYHZkiSuUYTq3nuWHKXnJenahdkijNHpcd1ms
nJZsm7LZqNAjDRB+q3Ki3XHGL7y5eESbbudknhjciiuoNmYh6ftAqeKSBRKmQylM
Bq/xX+g2M6E8a4UdWA/60gRIrRg7KHBUXcrvKV2KU/kZpa8OR9ZH+UkCgYEAx4Tm
NnLFuhWgiv9Wgzu5rjtCdiTUzurOfWxAh8r7d5vZHTO2qJD4gEk2BSFHDJcZa02U
M1zBh00apSGEgD4iOy+Lp35AcUTmooIWPFQRULCdzuo/Skk/mb2ct1mqVbKwtkMy
VyV4imCwzPN+mONmEB+VXIxf295tqtprDmCXGX0CgYEAqPNsfiu/HvIeHiZTSTj8
/MlheJ0vC2pXvLTxZD3PZUIDbb1N9C8IZDhEAlL3tsZ5W+RQjcSLalDKVA2CvAlr
WuVsP/SJevvSD71Wy/5p2ED2XpHpJWpFJTdJ4RhiUVyGkCRQHLjNaomHJohKk3wt
a0FD0LPNz46LcN106Fr8jwECgYEAhuGIhIyosTlHtFAUG1n4GBqFvrr9hvjkKZRS
N7r4r46Tg5NfS6vd41QbCfLKRm+rxofGxcZSKvbsKXB0VAItQBfPcKcAR9LNnFUX
VSd8ITGVLbncmYrVTUkLNkSOy6qmnkDlOlbhm6LsQ1HlZtRsPkAryEo5z7kaKKPK
NgkEfT0CgYA8fNoNvFED77WD6h7FzQDJLhMYmZOLzoIu3x4rH9SpJ9Q0GueiETA7
CLtMjrjzXUzj43Ksz7C91CDqD2fQQsY+7EWks2HL0zU2Or+O5/H/6yl/eJDTTd5a
CsWf+q7o+U7I3SBxLChc9G4Vy/AgZQJQES9CVAb6GOTY0jOIH7ZS7A==
-----END RSA PRIVATE KEY-----`)
)

func serve(conf *mainConfig, chand httputil.ContextHandler) {
	hand := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chand.ServeHTTP(context.Background(), w, r)
	})

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
