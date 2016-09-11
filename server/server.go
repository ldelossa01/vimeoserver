package server

import (
	"net/http"
	"vimeoserver/cache"
)

// VimeoService struct
type VimeoService struct {
	HTTPServer *http.Server
	httpClient *http.Client
	cache      cache.Cache
}

// NewVimeoService Get new instance
func NewVimeoService() *VimeoService {
	service := &VimeoService{
		httpClient: &http.Client{},
		cache:      cache.NewMemCache(64),
	}

	service.HTTPServer = &http.Server{Addr: "localhost:8000", Handler: createHandlers(service)}
	return service
}

// Attaches handlers to mux
func createHandlers(s *VimeoService) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.proxyRequest)
	return mux
}
