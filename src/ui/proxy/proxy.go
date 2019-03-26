package proxy

import (
	"fmt"
	"github.com/vmware/harbor/src/ui/config"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// Proxy is the instance of the reverse proxy in this package.
var Proxy *httputil.ReverseProxy
var Proxy2 *httputil.ReverseProxy

var handlers handlerChain

type handlerChain struct {
	head http.Handler
}

// Init initialize the Proxy instance and handler chain.
func Init(urls ...string) error {
	var err error
	var registryURL string
	if len(urls) > 1 {
		return fmt.Errorf("the parm, urls should have only 0 or 1 elements")
	}
	if len(urls) == 0 {
		registryURL, err = config.RegistryURL()
		if err != nil {
			return err
		}
	} else {
		registryURL = urls[0]
	}
	targetURL, err := url.Parse(registryURL)
	if err != nil {
		return err
	}
	Proxy = httputil.NewSingleHostReverseProxy(targetURL)
	if targetURL2 := os.Getenv("PROXY_REGISTRY_URL"); targetURL2 != "" {
		targetURL2, err := url.Parse(targetURL2)
		if err != nil {
			return err
		}
		Proxy2 = httputil.NewSingleHostReverseProxy(targetURL2)
	}

	//handlers = handlerChain{head: readonlyHandler{next: urlHandler{next: listReposHandler{next: contentTrustHandler{next: vulnerableHandler{next: Proxy}}}}}}
	handlers = handlerChain{head: readonlyHandler{next: urlHandler{next: listReposHandler{next: contentTrustHandler{next: vulnerableHandler{next: selectingProxyHandler{Proxy, Proxy2}}}}}}}
	return nil
}

// Handle handles the request.
func Handle(rw http.ResponseWriter, req *http.Request) {
	handlers.head.ServeHTTP(rw, req)
}
