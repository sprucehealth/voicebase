package main

import (
	"flag"
	"log"
	"net/http"
	"time"
	"os"
	"carefront/api"
)

var (
	flagListenAddr = flag.String("listen", ":8080", "Address and port to listen on")
)

const (
	CertKeyLocation string = "CERT_KEY"
	PrivateKeyLocation string = "PRIVATE_KEY"
	
)

func main() {
	flag.Parse()

	authApi := &api.MockAuth{
		Accounts: map[string]api.MockAccount{
			"fu": api.MockAccount{
				Id:       1,
				Password: "bar",
			},
		},
	}

	mux := &AuthServeMux{*http.NewServeMux(), authApi}

	authHandler := &AuthenticationHandler{authApi}
	pingHandler := PingHandler(0)
	photoHandler := &PhotoUploadHandler{api.PhotoService(0)}
	getSignedUrlsHandler :=  &GetSignedUrlsHandler{api.PhotoService(0)}

	mux.Handle("/v1/authenticate", authHandler)
	mux.Handle("/v1/ping", pingHandler)
	mux.Handle("/v1/upload", photoHandler)
	mux.Handle("/v1/imagesForCase/", getSignedUrlsHandler)
	
	s := &http.Server{
		Addr:           *flagListenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServeTLS(os.Getenv(CertKeyLocation), os.Getenv(PrivateKeyLocation)))
}
