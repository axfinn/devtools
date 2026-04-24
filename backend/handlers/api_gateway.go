package handlers

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultCPAGatewayBaseURL = "http://192.168.31.200:8317/v1"

type CPAProxyHandler struct {
	cpaProxy *httputil.ReverseProxy
}

func NewCPAProxyHandler() *CPAProxyHandler {
	target, err := url.Parse(defaultCPAGatewayBaseURL)
	if err != nil {
		panic("invalid CPA gateway target URL: " + err.Error())
	}

	const cpaPrefix = "/api/api-gateway/cpa/v1"
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		proxyPath := strings.TrimPrefix(req.URL.Path, cpaPrefix)
		if proxyPath == "/" {
			proxyPath = ""
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinURLPath(target.Path, proxyPath)
		req.URL.RawPath = req.URL.Path
		req.Host = target.Host

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}
	proxy.FlushInterval = -1
	proxy.Transport = &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp == nil {
			return nil
		}
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/event-stream") || resp.ContentLength < 0 {
			resp.Header.Del("Content-Length")
			resp.Header.Set("X-Accel-Buffering", "no")
			resp.Header.Set("Cache-Control", "no-cache, no-transform")
			resp.Header.Set("Connection", "keep-alive")
		}
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("cpa reverse proxy error: method=%s path=%s err=%v", req.Method, req.URL.Path, err)
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error":"下游 CPA 服务不可用"}`))
	}

	return &CPAProxyHandler{cpaProxy: proxy}
}

func (h *CPAProxyHandler) ProxyCPA(c *gin.Context) {
	h.cpaProxy.ServeHTTP(c.Writer, c.Request)
}

func joinURLPath(basePath, appendPath string) string {
	if appendPath == "" {
		if basePath == "" {
			return "/"
		}
		return basePath
	}

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
