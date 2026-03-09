package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultCPAGatewayBaseURL = "http://192.168.31.200:8317/v1"

type APIGatewayHandler struct {
	cpaProxy *httputil.ReverseProxy
}

func NewAPIGatewayHandler() *APIGatewayHandler {
	target, err := url.Parse(defaultCPAGatewayBaseURL)
	if err != nil {
		panic("invalid CPA gateway target URL: " + err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		proxyPath := strings.TrimPrefix(req.URL.Path, "/api/api-gateway/cpa/v1")
		if proxyPath == "" {
			proxyPath = "/"
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinURLPath(target.Path, proxyPath)
		req.URL.RawPath = req.URL.Path
		req.Host = target.Host

		// Mirror the standard reverse proxy behavior for User-Agent.
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}
	proxy.Transport = &http.Transport{
		Proxy: nil,
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error":"下游 CPA 服务不可用"}`))
	}

	return &APIGatewayHandler{cpaProxy: proxy}
}

func (h *APIGatewayHandler) ProxyCPA(c *gin.Context) {
	h.cpaProxy.ServeHTTP(c.Writer, c.Request)
}

func joinURLPath(basePath, appendPath string) string {
	baseHasSlash := strings.HasSuffix(basePath, "/")
	appendHasSlash := strings.HasPrefix(appendPath, "/")

	switch {
	case baseHasSlash && appendHasSlash:
		return basePath + strings.TrimPrefix(appendPath, "/")
	case !baseHasSlash && !appendHasSlash:
		return basePath + "/" + appendPath
	default:
		return basePath + appendPath
	}
}
