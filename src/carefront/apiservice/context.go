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

func GetContext(req *http.Request) *Context {
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
