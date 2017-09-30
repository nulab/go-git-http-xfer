package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/vvatanabe/go-git-http-transfer/addon/archivehandler"
	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

func main() {

	ght := githttptransfer.New("/data/git", "/usr/bin/git", true, true)

	ght.Event.On("prepare-service-rpc-upload", func(ctx githttptransfer.Context) error {
		log.Println("prepare run service rpc upload.")
		return nil
	})

	ght.Event.On("prepare-service-rpc-receive", func(ctx githttptransfer.Context) error {
		log.Println("prepare run service rpc receive.")
		return nil
	})

	// You can add some custom route.
	ght.AddRoute(githttptransfer.NewRoute(
		http.MethodGet,
		regexp.MustCompile("(.*?)/hello$"),
		func(ctx githttptransfer.Context) error {
			resp, req := ctx.Response(), ctx.Request()
			rp, fp := ctx.RepoPath(), ctx.FilePath()
			fmt.Fprintf(resp.Writer,
				"Hi there. URI: %s, RepoPath: %s, FilePath: %s",
				req.URL.RequestURI(), rp, fp)
			return nil
		},
	))

	// You can add some addon handler. (git archive)
	ght.AddRoute(githttptransfer.NewRoute(
		archivehandler.Method,
		archivehandler.Pattern,
		archivehandler.New(ght).HandlerFunc,
	))

	// You can add some middleware.
	handler := Logger(ght)

	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	})
}
