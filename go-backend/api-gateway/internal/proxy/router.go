// Package proxy provides the reverse proxy logic for the API Gateway.
// Routes incoming requests to the appropriate microservice based on URL prefix.
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"eticaret/shared/logger"
	"eticaret/shared/response"
)

// ServiceRoute defines a mapping from URL prefix to backend service.
type ServiceRoute struct {
	Prefix  string // e.g. "/auth"
	Target  string // e.g. "http://localhost:8081"
	NeedsJWT bool  // if true, gateway validates JWT before forwarding
}

// ReverseProxy creates a reverse proxy handler for a given target URL.
func ReverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		logger.Fatal("Invalid proxy target URL", "target", target, "error", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Custom error handler for when backend is unreachable
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Proxy error", "target", target, "path", r.URL.Path, "error", err)
		response.InternalServerError(w, "Servis şu an kullanılamıyor. Lütfen daha sonra tekrar deneyin.")
	}

	return proxy
}

// Router is the API Gateway's request dispatcher.
type Router struct {
	routes []ServiceRoute
	mux    *http.ServeMux
}

// NewRouter creates a configured gateway router.
func NewRouter(routes []ServiceRoute) *Router {
	router := &Router{routes: routes, mux: http.NewServeMux()}
	router.registerRoutes()
	return router
}

func (r *Router) registerRoutes() {
	for _, route := range r.routes {
		proxy := ReverseProxy(route.Target)
		prefix := route.Prefix + "/"

		logger.Info("Registering route", "prefix", prefix, "target", route.Target)
		r.mux.Handle(prefix, http.StripPrefix("", proxy))
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
