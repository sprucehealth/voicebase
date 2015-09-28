package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/regimens/internal/handlers"
	"github.com/sprucehealth/backend/libs/factual"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/products"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
)

var config struct {
	httpAddr      string
	proxyProtocol bool
	factualKey    string
	factualSecret string
}

func init() {
	flag.StringVar(&config.httpAddr, "http", "0.0.0.0:8000", "listen for http on `host:port`")
	flag.BoolVar(&config.proxyProtocol, "proxyproto", false, "enabled proxy protocol")
	flag.StringVar(&config.factualKey, "factual.key", "", "Factual API `key`")
	flag.StringVar(&config.factualSecret, "factual.secret", "", "Factual API `secret`")
}

func main() {
	log.SetFlags(log.Lshortfile)
	boot.ParseFlags("REGIMENS_")

	_, handler := setupRouter()

	serve(handler)
}

func setupRouter() (*mux.Router, httputil.ContextHandler) {
	productsSvc := &factualProductsService{cli: factual.New(config.factualKey, config.factualSecret)}

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/products", handlers.NewProducts(productsSvc))
	h := httputil.CompressResponse(httputil.DecompressRequest(router))
	return router, h
}

func serve(handler httputil.ContextHandler) {
	listener, err := net.Listen("tcp", config.httpAddr)
	if err != nil {
		log.Fatal(err)
	}
	if config.proxyProtocol {
		listener = &proxyproto.Listener{Listener: listener}
	}
	s := &http.Server{
		Handler:        httputil.FromContextHandler(handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.Serve(listener))
}

// TODO: this factual products service implementation is temporary to provide a useful stub

type factualProductsService struct {
	cli *factual.Client
}

func (s *factualProductsService) Search(query string) ([]*products.Product, error) {
	ps, err := s.cli.QueryProducts(query)
	if err != nil {
		return nil, err
	}
	prods := make([]*products.Product, len(ps))
	for i, p := range ps {
		prods[i] = &products.Product{
			ID:        p.FactualID,
			Name:      p.ProductName,
			ImageURLs: p.ImageURLs,
		}
	}
	return prods, nil
}
