# go-git-http-transfer [![Build Status](https://travis-ci.org/vvatanabe/go-git-http-transfer.svg?branch=master)](https://travis-ci.org/vvatanabe/go-git-http-transfer) [![Coverage Status](https://coveralls.io/repos/github/vvatanabe/go-git-http-transfer/badge.svg?branch=master)](https://coveralls.io/github/vvatanabe/go-git-http-transfer?branch=master)

Implements Git HTTP Transport.

## Support Protocol

* The Smart Protocol
* The Dumb Protocol

## Requires

* Go 1.7+

## Quickly Trial

Let's clone this repository and execute the following commands.

```` zsh
$ docker build -t git-http-transfer-build .
$ docker run -it --rm -v $PWD:/go/src/github.com/vvatanabe/go-git-http-transfer \
    -p 8080:8080 git-http-transfer-build bash

# in container
$ go run /go/src/github.com/vvatanabe/go-git-http-transfer/example/main.go -p 8080

# in your local machine
$ git clone http://localhost:8080/example.git
````

## Installation

This package can be installed with the go get command:

``` zsh
$ go get github.com/vvatanabe/go-git-http-transfer
```

## Usage

Basic
``` go
package main

import (
	"log"
	"net/http"

	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

func main() {
	
	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	
	if err := http.ListenAndServe(":8080", ght); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```
You can add optional parameters to constructor function.
* `WithoutUploadPack`: Disabled `git-upload-pack` command.
* `WithoutReceivePack`: Disabled `git-receive-pack` command.
* `WithoutDumbProto`: Disabled `dumb protocol` handling.
```go
	ght, err := githttptransfer.New(
		"/data/git",
		"/usr/bin/git",
		githttptransfer.WithoutUploadPack(),
		githttptransfer.WithoutReceivePack(),
		githttptransfer.WithoutDumbProto(),
	)
```
You can add some custom route.
``` go
func main() {

	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}

	// You can add some custom route.
	ght.Router.Add(githttptransfer.NewRoute(
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
	
	if err := http.ListenAndServe(":8080", ght); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```
You can add some middleware.
``` go
func main() {
	
	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	
	handler := Logging(ght)
	
	if err := http.ListenAndServe(":8080", ght); err != nil {
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
	"github.com/vvatanabe/go-git-http-transfer/addon/archivehandler"
)

func main() {
	ght, err := githttptransfer.New("/data/git", "/usr/bin/git")
	if err != nil {
		log.Fatalf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	
	ght.Router.Add(githttptransfer.NewRoute(
		archivehandler.Method,
		archivehandler.Pattern,
		archivehandler.New(ght).HandlerFunc,
	))
	
	if err := http.ListenAndServe(":8080", ght); err != nil {
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
