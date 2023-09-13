package steelhead

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var Errors = make(chan Error)

func ErrorHandler(f func(Error)) {
	go func() {
		for err := range Errors {
			f(err)
		}
	}()
}

func chain(f Handler, tower ...Middleware) http.HandlerFunc {
	for _, middleware := range tower {
		f = middleware(f)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s, b, e := f(r)
		if e != nil { Errors <- Error{e, r} }
		handleResponse(w, s, b, e)
	};
}

func Middlewares(mws ...Middleware) []Middleware {
	return mws
}

func Handlers(vargs ...any) map[string]Handler {
	var even = func(n int) bool { return n%2 == 0 }
	if !even(len(vargs)) {
		log.Fatalf("args must be even %+v", vargs)
	}
	var res = make(map[string]Handler)
	for i := 0; i < len(vargs); i += 2 {
		res[vargs[i].(string)] = vargs[i+1].(Handler)
	}
	return res
}

func Children(vargs ...any) map[string]Router {
	var even = func(n int) bool { return n%2 == 0 }
	if !even(len(vargs)) {
		log.Fatalf("Children args must be even")
	}
	var res = make(map[string]Router)
	for i := 0; i < len(vargs); i += 2 {
		res[vargs[i].(string)] = vargs[i+1].(Router)
	}
	return res
}

func Node(vargs ...any) Router {
	return Router{
		Middlewares(),
		Handlers(),
		Children(vargs...),
	}
}

func WrapNode(middlewares []Middleware, vargs ...any) Router {
	return Router{
		middlewares,
		Handlers(),
		Children(vargs...),
	}
}

func ExtNode(handlers map[string]Handler, vargs ...any) Router {
	return Router{
		Middlewares(),
		handlers,
		Children(vargs...),
	}
}

func WrapExtNode(middlewares []Middleware, handlers map[string]Handler, vargs ...any) Router {
	return Router{
		middlewares,
		handlers,
		Children(vargs...),
	}
}

func Leaf(vargs ...any) Router {
	return Router{
		Middlewares(),
		Handlers(vargs...),
		Children(),
	}
}

func WrapLeaf(middlewares []Middleware, vargs ...any) Router {
	return Router{
		Middlewares(middlewares...),
		Handlers(vargs...),
		Children(),
	}
}

func BuildRouter(tree Router) *mux.Router {
	r := mux.NewRouter()
	var recur func(router *mux.Router, middlewares []Middleware, path string, tree Router)
	recur = func(router *mux.Router, middlewares []Middleware, path string, tree Router) {
		for _, mw := range tree.Middlewares {
			middlewares = append(middlewares, mw)
		}

		for k, c := range tree.Children {
			recur(router, middlewares, fmt.Sprintf("%s%s", path, k), c)
		}

		// cringe
		for i, j := 0, len(middlewares)-1; i < j; i, j = i+1, j-1 {
			middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
		}

		for k, h := range tree.Handlers {
			router.HandleFunc(path, chain(h, middlewares...)).Methods(k)
		}
	}
	recur(r, Middlewares(), "", tree)
	return r
}
