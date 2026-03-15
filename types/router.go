package types

// RouterInfo holds routing information for the web server
type RouterInfo struct {
	BasePath    string
	Router      *gorilla.Mux
	Middleware  []func(http.Handler) http.Handler
}
