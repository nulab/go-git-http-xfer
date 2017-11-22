package githttptransfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	serviceRPCUpload = func(path string) (match string) {
		return hasSuffix(path, "/git-upload-pack")
	}
	serviceRPCReceive = func(path string) (match string) {
		return hasSuffix(path, "/git-receive-pack")
	}

	getInfoRefs = func(path string) (match string) {
		return hasSuffix(path, "/info/refs")
	}

	getHead = func(path string) (match string) {
		return hasSuffix(path, "/HEAD")
	}

	getAlternates = func(path string) (match string) {
		return hasSuffix(path, "/objects/info/alternates")
	}

	getHTTPAlternates = func(path string) (match string) {
		return hasSuffix(path, "/objects/info/http-alternates")
	}

	getInfoPacks = func(path string) (match string) {
		return hasSuffix(path, "/objects/info/packs")
	}

	getInfoFile = func(path string) (match string) {
		return findStringSubmatch(path, regexp.MustCompile(".*?(/objects/info/[^/]*)$"))
	}

	getLooseObject = func(path string) (match string) {
		return findStringSubmatch(path, regexp.MustCompile(".*?(/objects/[0-9a-f]{2}/[0-9a-f]{38})$"))
	}

	getPackFile = func(path string) (match string) {
		return findStringSubmatch(path, regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.pack)$"))
	}

	getIdxFile = func(path string) (match string) {
		return findStringSubmatch(path, regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.idx)$"))
	}
)

func hasSuffix(path, suffix string) (match string) {
	if strings.HasSuffix(path, suffix) {
		match = suffix
	}
	return
}

func findStringSubmatch(path string, prefix *regexp.Regexp) (match string) {
	if m := prefix.FindStringSubmatch(path); m != nil {
		match = m[1]
	}
	return
}

type options struct {
	uploadPack  bool
	receivePack bool
	dumbProto   bool
}

type Option func(*options)

func WithoutUploadPack() Option {
	return func(o *options) {
		o.uploadPack = false
	}
}

func WithoutReceivePack() Option {
	return func(o *options) {
		o.receivePack = false
	}
}

func WithoutDumbProto() Option {
	return func(o *options) {
		o.dumbProto = false
	}
}

func New(gitRootPath, gitBinPath string, opts ...Option) (*GitHTTPTransfer, error) {

	if gitRootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		gitRootPath = cwd
	}

	ghtOpts := &options{true, true, true}

	for _, opt := range opts {
		opt(ghtOpts)
	}

	git := newGit(gitRootPath, gitBinPath, ghtOpts.uploadPack, ghtOpts.receivePack)
	router := newRouter()
	event := newEvent()

	ght := &GitHTTPTransfer{git, router, event}

	ght.Router.Add(NewRoute(http.MethodPost, serviceRPCUpload, ght.serviceRPCUpload))

	ght.Router.Add(NewRoute(http.MethodPost, serviceRPCReceive, ght.serviceRPCReceive))
	ght.Router.Add(NewRoute(http.MethodGet, getInfoRefs, ght.getInfoRefs))

	if ghtOpts.dumbProto {
		ght.Router.Add(NewRoute(http.MethodGet, getHead, ght.getTextFile))
		ght.Router.Add(NewRoute(http.MethodGet, getAlternates, ght.getTextFile))
		ght.Router.Add(NewRoute(http.MethodGet, getHTTPAlternates, ght.getTextFile))
		ght.Router.Add(NewRoute(http.MethodGet, getInfoPacks, ght.getInfoPacks))
		ght.Router.Add(NewRoute(http.MethodGet, getInfoFile, ght.getTextFile))
		ght.Router.Add(NewRoute(http.MethodGet, getLooseObject, ght.getLooseObject))
		ght.Router.Add(NewRoute(http.MethodGet, getPackFile, ght.getPackFile))
		ght.Router.Add(NewRoute(http.MethodGet, getIdxFile, ght.getIdxFile))
	}
	return ght, nil
}

type GitHTTPTransfer struct {
	Git    *git
	Router *router
	Event  *event
}

func (ght *GitHTTPTransfer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	repoPath, filePath, handler, err := ght.matchRouting(r.Method, r.URL.Path)
	switch err.(type) {
	case *URLNotFoundError:
		RenderNotFound(rw)
		return
	case *MethodNotAllowedError:
		RenderMethodNotAllowed(rw, r)
		return
	}

	ctx := NewContext(rw, r, repoPath, filePath)

	ght.Event.emit(AfterMatchRouting, ctx)

	if !ght.Git.Exists(ctx.RepoPath()) {
		RenderNotFound(ctx.Response().Writer)
		return
	}

	handler(ctx)
}

func (ght *GitHTTPTransfer) matchRouting(method, path string) (repoPath string, filePath string, handler HandlerFunc, err error) {
	match, route, err := ght.Router.Match(method, path)
	if err == nil {
		repoPath = strings.Replace(path, match, "", 1)
		filePath = strings.Replace(path, repoPath+"/", "", 1)
		handler = route.Handler
	}
	return
}

const (
	uploadPack  = "upload-pack"
	receivePack = "receive-pack"
)

type HandlerFunc func(ctx Context)

func newEvent() *event {
	return &event{map[EventKey]HandlerFunc{}}
}

type EventKey string

const (
	PrepareServiceRPCUpload  EventKey = "prepare-service-rpc-upload"
	PrepareServiceRPCReceive EventKey = "prepare-service-rpc-receive"
	AfterMatchRouting        EventKey = "after-match-routing"
)

type event struct {
	listeners map[EventKey]HandlerFunc
}

func (e *event) emit(evt EventKey, ctx Context) {
	v, ok := e.listeners[evt]
	if ok {
		v(ctx)
	}
}

func (e *event) On(evt EventKey, listener HandlerFunc) {
	e.listeners[evt] = listener
}

func (ght *GitHTTPTransfer) serviceRPCUpload(ctx Context) {
	ght.Event.emit(PrepareServiceRPCUpload, ctx)
	ght.serviceRPC(ctx, uploadPack)
}

func (ght *GitHTTPTransfer) serviceRPCReceive(ctx Context) {
	ght.Event.emit(PrepareServiceRPCReceive, ctx)
	ght.serviceRPC(ctx, receivePack)
}

func (ght *GitHTTPTransfer) serviceRPC(ctx Context, rpc string) {

	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	if !ght.Git.HasAccess(req, rpc, true) {
		RenderNoAccess(ctx.Response().Writer)
		return
	}

	var body io.ReadCloser
	var err error

	if req.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(req.Body)
		if err != nil {
			RenderInternalServerError(ctx.Response().Writer)
			return
		}
	} else {
		body = req.Body
	}
	defer body.Close()

	res.SetContentType(fmt.Sprintf("application/x-git-%s-result", rpc))

	args := []string{rpc, "--stateless-rpc", "."}
	cmd := ght.Git.GitCommand(repoPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		RenderInternalServerError(ctx.Response().Writer)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		RenderInternalServerError(ctx.Response().Writer)
		return
	}

	if err = cmd.Start(); err != nil {
		RenderInternalServerError(ctx.Response().Writer)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer stdin.Close()
		io.Copy(stdin, body)
	}()

	go func() {
		defer wg.Done()
		defer stdout.Close()
		res.Copy(stdout)
	}()

	wg.Wait()

	if err = cmd.Wait(); err != nil {
		RenderInternalServerError(ctx.Response().Writer)
		return
	}
}

func (ght *GitHTTPTransfer) getInfoRefs(ctx Context) {
	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	serviceName := getServiceType(req)
	if !ght.Git.HasAccess(req, serviceName, false) {
		ght.Git.UpdateServerInfo(repoPath)
		res.HdrNocache()
		if err := ght.sendFile("text/plain; charset=utf-8", ctx); err != nil {
			RenderNotFound(ctx.Response().Writer)
		}
	}

	refs, err := ght.Git.GetInfoRefs(repoPath, serviceName)
	if err != nil {
		RenderNotFound(ctx.Response().Writer)
		return
	}

	res.HdrNocache()
	res.SetContentType(fmt.Sprintf("application/x-git-%s-advertisement", serviceName))
	res.WriteHeader(http.StatusOK)
	res.PktWrite("# service=git-" + serviceName + "\n")
	res.PktFlush()
	res.Write(refs)
}

func (ght *GitHTTPTransfer) getInfoPacks(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ght.sendFile("text/plain; charset=utf-8", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ght *GitHTTPTransfer) getLooseObject(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ght.sendFile("application/x-git-loose-object", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ght *GitHTTPTransfer) getPackFile(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ght.sendFile("application/x-git-packed-objects", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}

}

func (ght *GitHTTPTransfer) getIdxFile(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ght.sendFile("application/x-git-packed-objects-toc", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ght *GitHTTPTransfer) getTextFile(ctx Context) {
	ctx.Response().HdrNocache()
	if err := ght.sendFile("text/plain", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ght *GitHTTPTransfer) sendFile(contentType string, ctx Context) error {
	res, req, repoPath, filePath := ctx.Response(), ctx.Request(), ctx.RepoPath(), ctx.FilePath()
	fileInfo, err := ght.Git.GetRequestFileInfo(repoPath, filePath)
	if err != nil {
		return err
	}
	res.SetContentType(contentType)
	res.SetContentLength(fmt.Sprintf("%d", fileInfo.Size()))
	res.SetLastModified(fileInfo.ModTime().Format(http.TimeFormat))
	http.ServeFile(res.Writer, req, fileInfo.AbsolutePath)
	return nil
}
