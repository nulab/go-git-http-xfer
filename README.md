# go-git-http-xfer [![Build Status](https://travis-ci.org/nulab/go-git-http-xfer.svg?branch=master)](https://travis-ci.org/nulab/go-git-http-xfer) [![Coverage Status](https://coveralls.io/repos/github/nulab/go-git-http-xfer/badge.svg?branch=master)](https://coveralls.io/github/nulab/go-git-http-xfer?branch=master)

Implements Git HTTP Transport.

## Support Protocol

* The Smart Protocol
* The Dumb Protocol

## Requires

* Go 1.7+

## Quickly Trial

Let's clone this repository and execute the following commands.

```` zsh
# docker build
$ docker build -t git-http-xfer .

# test
$ docker run --rm -v $PWD:/go/src/github.com/nulab/go-git-http-xfer go-git-http-xfer \
    bash -c "gotestcover -v -covermode=count -coverprofile=coverage.out ./..."

# run server
$ docker run -it --rm -v $PWD:/go/src/github.com/nulab/go-git-http-xfer -p 5050:5050 go-git-http-xfer \
    bash -c "go run ./example/main.go -p 5050"

# in your local machine
$ git clone http://localhost:5050/example.git
````

## Installation

This package can be installed with the go get command:

``` zsh
$ go get github.com/nulab/go-git-http-xfer
```

## Usage

Basic
``` go
package main

import (
	"log"
	"net/http"

	"github.com/nulab/go-git-http-xfer/githttpxfer"
)

func main() {
	
	ghx, err := githttpxfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}
	
	if err := http.ListenAndServe(":5050", ghx); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```
You can add optional parameters to constructor function.
* `DisableUploadPack` : Disable `git upload-pack` command.
* `DisableReceivePack`: Disable `git receive-pack` command.
* `WithoutDumbProto`  : Without `dumb protocol` handling.
```go
	ghx, err := githttpxfer.New(
		"/data/git",
		"/usr/bin/git",
		githttpxfer.DisableUploadPack(),
		githttpxfer.DisableReceivePack(),
		githttpxfer.WithoutDumbProto(),
	)
```
You can add some custom route.
``` go
func main() {

	ghx, err := githttpxfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}

	// You can add some custom route.
	ghx.Router.Add(githttpxfer.NewRoute(
		http.MethodGet,
		func (path string) (match string) {
			suffix := "/hello"
			if strings.HasSuffix(path, suffix) {
				match = suffix
			}
			return
		},
		func(ctx githttpxfer.Context) {
			resp, req := ctx.Response(), ctx.Request()
			rp, fp := ctx.RepoPath(), ctx.FilePath()
			fmt.Fprintf(resp.Writer,
				"Hi there. URI: %s, RepoPath: %s, FilePath: %s",
				req.URL.RequestURI(), rp, fp)
		},
	))
	
	if err := http.ListenAndServe(":5050", ghx); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```
You can add some middleware.
``` go
func main() {
	
	ghx, err := githttpxfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}
	
	handler := Logging(ghx)
	
	if err := http.ListenAndServe(":5050", ghx); err != nil {
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
```
You can add some addon handler. (git archive)
``` go
import (
	"github.com/nulab/go-git-http-xfer/addon/handler/archive"
)

func main() {
	ghx, err := githttpxfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}
	
	ghx.Router.Add(githttpxfer.NewRoute(
		archive.Method,
		archive.Pattern,
		archive.New(ghx).Archive,
	))
	
	if err := http.ListenAndServe(":5050", ghx); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

```

## Reference

- [Git Internals - Transfer Protocols](http://www.opensource.org/licenses/mit-license.php)
- [grackorg/grack](https://github.com/grackorg/grack)
- [dragon3/Plack-App-GitSmartHttp](https://github.com/dragon3/Plack-App-GitSmartHttp)

## Bugs and Feedback

For bugs, questions and discussions please use the Github Issues.

## License

[MIT License](http://www.opensource.org/licenses/mit-license.php)
