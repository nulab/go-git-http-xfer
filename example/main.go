package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"flag"

	"github.com/vvatanabe/go-git-http-transfer/addon/archivehandler"
	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
	"strings"
)

func main() {

	var port int
	flag.IntVar(&port, "p", 5050, "port of git httpd server.")
	flag.Parse()

	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPTransfer instance could not be created. %s", err.Error())
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
		archivehandler.Method,
		archivehandler.Pattern,
		archivehandler.New(ght).HandlerFunc,
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
