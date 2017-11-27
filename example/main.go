package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"flag"

	"strings"

	"github.com/vvatanabe/go-git-http-transfer/addon/handler/archive"
	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

func main() {

	var port int
	flag.IntVar(&port, "p", 5050, "port of git httpd server.")
	flag.Parse()

	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatal("GitHTTPTransfer instance could not be created.", err)
		return
	}

	ght.Event.On(githttptransfer.PrepareServiceRPCUpload, func(ctx githttptransfer.Context) {
		// prepare run service rpc upload.
	})

	ght.Event.On(githttptransfer.PrepareServiceRPCReceive, func(ctx githttptransfer.Context) {
		// prepare run service rpc receive.
	})

	ght.Event.On(githttptransfer.AfterMatchRouting, func(ctx githttptransfer.Context) {
		// after match routing.
	})

	// You can add some custom route.
	ght.Router.Add(githttptransfer.NewRoute(
		http.MethodGet,
		func(path string) (match string) {
			suffix := "/hello"
			if strings.HasSuffix(path, suffix) {
				match = suffix
			}
			return
		},
		func(ctx githttptransfer.Context) {
			resp, req := ctx.Response(), ctx.Request()
			rp, fp := ctx.RepoPath(), ctx.FilePath()
			fmt.Fprintf(resp.Writer,
				"Hi there. URI: %s, RepoPath: %s, FilePath: %s",
				req.URL.RequestURI(), rp, fp)
		},
	))

	// You can add some addon handler. (git archive)
	ght.Router.Add(githttptransfer.NewRoute(
		archive.Method,
		archive.Pattern,
		archive.New(ght).Archive,
	))

	// You can add some middleware.
	handler := Logging(ght)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	})
}

func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "nulab" || password != "DeaDBeeF" {
			RenderUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RenderUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Please enter your username and password."`)

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
	w.Header().Set("Content-Type", "text/plain")
}
