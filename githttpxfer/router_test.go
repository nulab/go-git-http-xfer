package githttpxfer

import (
	"net/http"
	"net/url"
	"testing"
)

func Test_Router_Append_should_append_route(t *testing.T) {
	router := &router{}
	router.Add(&Route{
		http.MethodPost,
		func(u *url.URL) *Match {
			return matchSuffix(u.Path, "/foo")
		},
		func(ctx Context) {},
	})
	router.Add(&Route{
		http.MethodPost,
		func(u *url.URL) *Match {
			return matchSuffix(u.Path, "/bar")
		},
		func(ctx Context) {},
	})
	length := len(router.routes)
	expected := 2
	if expected != length {
		t.Errorf("router length is not %d . result: %d", expected, length)
	}
}

func Test_Router_Match_should_match_route(t *testing.T) {
	router := &router{}
	router.Add(&Route{
		http.MethodPost,
		func(u *url.URL) *Match {
			return matchSuffix(u.Path, "/foo")
		},
		func(ctx Context) {},
	})
	match, route, err := router.Match(http.MethodPost, &url.URL{Path: "/base/foo"})
	if err != nil {
		t.Errorf("error is %s", err.Error())
	}
	if http.MethodPost != route.Method {
		t.Errorf("http method is not %s . result: %s", http.MethodPost, route.Method)
	}
	if "foo" != match.FilePath {
		t.Errorf("match is not %s . result: %s", "foo", match.FilePath)
	}
}

func Test_Router_Match_should_return_UrlNotFound_error(t *testing.T) {
	router := &router{}
	router.Add(&Route{
		http.MethodPost,
		func(u *url.URL) *Match {
			return matchSuffix(u.Path, "/foo")
		},
		func(ctx Context) {},
	})
	match, route, err := router.Match(http.MethodPost, &url.URL{Path: "/base/hoge"})
	if err == nil {
		t.Error("error is nil.")
	}
	if match != nil {
		t.Error("match is not empty.")
	}
	if route != nil {
		t.Error("route is not nil.")
	}
	switch err.(type) {
	case *URLNotFoundError:
		return
	}
	t.Errorf("error is not UrlNotFound. %s", err.Error())
}

func Test_Router_Match_should_return_MethodNotAllowed_error(t *testing.T) {
	router := &router{}
	router.Add(&Route{
		http.MethodPost,
		func(u *url.URL) *Match {
			return matchSuffix(u.Path, "/foo")
		},
		func(ctx Context) {},
	})
	match, route, err := router.Match(http.MethodGet, &url.URL{Path: "/base/foo"})
	if err == nil {
		t.Error("error is nil.")
	}
	if match != nil {
		t.Error("match is not empty.")
	}
	if route != nil {
		t.Error("route is not nil.")
	}
	if _, is := err.(*MethodNotAllowedError); !is {
		t.Errorf("error is not MethodNotAllowed. %s", err.Error())
		return
	}
}
