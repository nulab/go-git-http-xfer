package githttptransfer

import (
	"regexp"
)

type router struct {
	routes []*Route
}

func (r *router) add(route *Route) {
	if r.routes == nil {
		r.routes = []*Route{}
	}
	r.routes = append(r.routes, route)
}

func (r *router) match(method string, path string) (match []string, route *Route, err error) {
	for _, v := range r.routes {
		if m := v.Pattern.FindStringSubmatch(path); m != nil {
			if v.Method != method {
				err = &MethodNotAllowedError{
					Method: method,
					Path:   path,
				}
				return nil, nil, err // no need to return params?
			}
			match = m
			route = v
			return match, route, nil // no need to return params?
		}
	}
	err = &UrlNotFoundError{
		Method: method,
		Path:   path,
	}
	return nil, nil, err // no need to return params?
}

func newRouter() *router {
	return &router{routes: []*Route{}}
}

type Route struct {
	Method  string
	Pattern *regexp.Regexp
	Handler HandlerFunc
}

func NewRoute(method string, pattern *regexp.Regexp, handler HandlerFunc) *Route {
	return &Route{method, pattern, handler}
}
