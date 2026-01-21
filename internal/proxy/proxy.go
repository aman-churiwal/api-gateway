package proxy

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Proxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func New(targetURL string) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		target: target,
		proxy:  httputil.NewSingleHostReverseProxy(target),
	}, nil
}

func (p *Proxy) Handle(c *gin.Context) {
	c.Request.URL.Host = p.target.Host
	c.Request.URL.Scheme = p.target.Scheme
	c.Request.Header.Set("X-Forwarded-Host", c.Request.Header.Get("Host"))
	c.Request.Host = p.target.Host

	if clientIP := c.ClientIP(); clientIP != "" {
		c.Request.Header.Set("X-Forwarded-For", clientIP)
	}

	p.proxy.ServeHTTP(c.Writer, c.Request)
}
