package githttpxfer

import "net/url"

type router struct {
	routes []*Route
}

func (r *router) Add(route *Route) {
	if r.routes == nil {
		r.routes = []*Route{}
	}
	r.routes = append(r.routes, route)
}

func (r *router) Match(method string, u *url.URL) (match *Match, route *Route, err error) {
	for _, v := range r.routes {
		if m := v.Pattern(u); m != nil {
			if v.Method != method {
				err = &MethodNotAllowedError{
					Method: method,
					Path:   u.Path,
				}
				return
			}
			match = m
			route = v
			return
		}
	}

	err = &URLNotFoundError{
		Method: method,
		Path:   u.Path,
	}
	return
}

func newRouter() *router {
	return &router{routes: []*Route{}}
}

type Pattern = func(u *url.URL) *Match

type Route struct {
	Method  string
	Pattern Pattern
	Handler HandlerFunc
}

func NewRoute(method string, pattern Pattern, handler HandlerFunc) *Route {
	return &Route{method, pattern, handler}
}
