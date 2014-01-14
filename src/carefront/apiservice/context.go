package apiservice

import (
	"net/http"
	"sync"
)

var (
	ctxMu          sync.Mutex
	requestContext = map[*http.Request]*Context{}
)

type Context struct {
	AccountId int64
}

// TODO: During testing this is the context that's returned for any request. This is necessary because at
// the moment there's no great way to pass around things like AccountId. The reason using this global is
// bad is that it doesn't allow for parallel tests.
var TestingContext = &Context{}

func GetContext(req *http.Request) *Context {
	if Testing {
		return TestingContext
	}
	ctxMu.Lock()
	defer ctxMu.Unlock()
	if ctx := requestContext[req]; ctx != nil {
		return ctx
	}
	ctx := &Context{}
	requestContext[req] = ctx
	return ctx
}

func DeleteContext(req *http.Request) {
	ctxMu.Lock()
	delete(requestContext, req)
	ctxMu.Unlock()
}
