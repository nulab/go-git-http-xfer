package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

func main() {

	ght := githttptransfer.New("/data/git", "/usr/bin/git", true, true)

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
