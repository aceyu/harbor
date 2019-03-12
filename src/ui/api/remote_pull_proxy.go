package api

import (
	"crypto/tls"
	"fmt"
	"github.com/astaxie/beego/context"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/secret"
	"github.com/vmware/harbor/src/common/utils/log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var proxy *httputil.ReverseProxy

func InitRemotePullProxy() {
	targetURL, err := url.Parse(singlePullReplicationUrl())
	if err == nil {
		proxy = httputil.NewSingleHostReverseProxy(targetURL)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		proxy.Transport = &remotePullTransport{
			transport: tr,
		}
		targetQuery := targetURL.RawQuery
		director := func(req *http.Request) {
			req.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = singleJoiningSlash(targetURL.Path, req.URL.Path)
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
		}
		proxy.Director = director
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func RemotePullProxy(ctx *context.Context) {
	var user models.User
	userInterface := ctx.Input.Session("user")

	if userInterface == nil {
		log.Debug("can not get user information from session")
		return
	}

	log.Debug("got user information from session")
	user, ok := userInterface.(models.User)
	if !ok {
		log.Info("can not get user information from session")
		return
	}
	ctx.Request.URL.User = url.User(user.Username)

	proxy.ServeHTTP(ctx.ResponseWriter, ctx.Request)
}

type remotePullTransport struct {
	transport http.RoundTripper
}

func (t *remotePullTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("IS-REMOTE-PULL", "true")

	username := ""
	if req.URL.User != nil {
		username = req.URL.User.Username()
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s%s@%s", secret.HeaderPrefix, secretForRemotePull(), username))
	resp, err := t.transport.RoundTrip(req)
	if err == nil {
		//resp
		resp.Header.Del("Set-Cookie")
	}
	return resp, err
}

func secretForRemotePull() string {
	return os.Getenv("SECRET_FOR_REMOTE_PULL")
}

func singlePullReplicationUrl() string {
	return os.Getenv("SINGLE_PULL_REPLICATION_URL")
}
