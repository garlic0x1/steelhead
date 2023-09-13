package steelhead

import "net/http"

type Handler func(*http.Request) (int, any, error)
type Middleware func(Handler) Handler

type PageQuery struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type Page struct {
	Count int `json:"count"`
	Data  any `json:"data"`
}

type HttpError struct {
	Status int   `json:"status"`
	Error string `json:"error"`
}

type Router struct {
	Middlewares []Middleware
	Handlers    map[string]Handler
	Children    map[string]Router
}
