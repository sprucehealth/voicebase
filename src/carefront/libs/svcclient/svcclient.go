package svcclient

import (
	"errors"
	"net/rpc"
	"sync"

	"github.com/samuel/go-metrics/metrics"
)

type ClientBuilder interface {
	NewClient() (*rpc.Client, error)
}

type Client struct {
	clientId           string
	mu                 sync.RWMutex // protects clients and closed
	closed             bool
	clients            []*rpc.Client
	maxIdleConnections int
	clientBuilder      ClientBuilder

	statRequests               metrics.Counter
	statEstablishedConnections metrics.Counter
	statLiveConnections        metrics.Counter
}

func NewClient(clientId string, maxIdleConnections int, clientBuilder ClientBuilder, metricsRegistry metrics.Registry) *Client {
	r := &Client{
		clientId:           clientId,
		clients:            make([]*rpc.Client, 0, maxIdleConnections),
		maxIdleConnections: maxIdleConnections,
		clientBuilder:      clientBuilder,
		closed:             false,

		statRequests:               metrics.NewCounter(),
		statEstablishedConnections: metrics.NewCounter(),
		statLiveConnections:        metrics.NewCounter(),
	}

	metricsRegistry.Add("requests", r.statRequests)
	metricsRegistry.Add("connections.established", r.statEstablishedConnections)
	metricsRegistry.Add("connections.inuse", r.statLiveConnections)

	return r
}

// conn returns a cached or newly-opened *rpc.Client
func (r *Client) conn() (*rpc.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, errors.New("svcclient: connection is closed")
	}
	if n := len(r.clients); n > 0 {
		c := r.clients[n-1]
		r.clients = r.clients[:n-1]
		return c, nil
	}
	r.statEstablishedConnections.Inc(1)
	return r.clientBuilder.NewClient()
}

// putConn adds a connection to the free pool
func (r *Client) putConn(c *rpc.Client) {
	r.mu.Lock()
	if n := len(r.clients); !r.closed && n < r.maxIdleConnections {
		r.clients = append(r.clients, c)
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	c.Close()
}

func (r *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {
	c, err := r.conn()
	if err != nil {
		return err
	}
	r.statLiveConnections.Inc(1)
	r.statRequests.Inc(1)
	err = c.Call(serviceMethod, args, reply)
	r.statLiveConnections.Dec(1)
	if err != nil {
		c.Close()
		return err
	}
	r.putConn(c)
	return nil
}

func (r *Client) Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) *rpc.Call {
	if done == nil {
		panic("svcclient: method Go requires a non-nil done chan")
	}
	c, err := r.conn()
	if err != nil {
		call := &rpc.Call{
			ServiceMethod: serviceMethod,
			Args:          args,
			Reply:         reply,
			Error:         err,
			Done:          done,
		}
		done <- call
		return call
	}
	r.statLiveConnections.Inc(1)
	d := make(chan *rpc.Call, 1)
	go func() {
		call := <-d
		call.Done = done
		r.statLiveConnections.Dec(1)
		if call.Error != nil {
			c.Close()
		} else {
			r.putConn(c)
		}
		done <- call
	}()
	r.statRequests.Inc(1)
	return c.Go(serviceMethod, args, reply, d)
}

func (r *Client) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var err error
	for _, c := range r.clients {
		err1 := c.Close()
		if err1 != nil {
			err = err1
		}
	}
	r.clients = nil
	r.closed = true
	return err
}
