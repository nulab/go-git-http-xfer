package githttpxfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
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

	getInfoFileRegexp = regexp.MustCompile(".*?(/objects/info/[^/]*)$")
	getInfoFile       = func(path string) (match string) {
		return findStringSubmatch(path, getInfoFileRegexp)
	}

	getLooseObjectRegexp = regexp.MustCompile(".*?(/objects/[0-9a-f]{2}/[0-9a-f]{38})$")
	getLooseObject       = func(path string) (match string) {
		return findStringSubmatch(path, getLooseObjectRegexp)
	}

	getPackFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.pack)$")
	getPackFile       = func(path string) (match string) {
		return findStringSubmatch(path, getPackFileRegexp)
	}

	getIdxFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.idx)$")
	getIdxFile       = func(path string) (match string) {
		return findStringSubmatch(path, getIdxFileRegexp)
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

func DisableUploadPack() Option {
	return func(o *options) {
		o.uploadPack = false
	}
}

func DisableReceivePack() Option {
	return func(o *options) {
		o.receivePack = false
	}
}

func WithoutDumbProto() Option {
	return func(o *options) {
		o.dumbProto = false
	}
}

func New(gitRootPath, gitBinPath string, opts ...Option) (*GitHTTPXfer, error) {

	if gitRootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		gitRootPath = cwd
	}

	ghxOpts := &options{true, true, true}

	for _, opt := range opts {
		opt(ghxOpts)
	}

	git := newGit(gitRootPath, gitBinPath, ghxOpts.uploadPack, ghxOpts.receivePack)
	router := newRouter()
	event := newEvent()

	ghx := &GitHTTPXfer{git, router, event, &defaultLogger{}}

	ghx.Router.Add(NewRoute(http.MethodPost, serviceRPCUpload, ghx.serviceRPCUpload))
	ghx.Router.Add(NewRoute(http.MethodPost, serviceRPCReceive, ghx.serviceRPCReceive))
	ghx.Router.Add(NewRoute(http.MethodGet, getInfoRefs, ghx.getInfoRefs))

	if ghxOpts.dumbProto {
		ghx.Router.Add(NewRoute(http.MethodGet, getHead, ghx.getTextFile))
		ghx.Router.Add(NewRoute(http.MethodGet, getAlternates, ghx.getTextFile))
		ghx.Router.Add(NewRoute(http.MethodGet, getHTTPAlternates, ghx.getTextFile))
		ghx.Router.Add(NewRoute(http.MethodGet, getInfoPacks, ghx.getInfoPacks))
		ghx.Router.Add(NewRoute(http.MethodGet, getInfoFile, ghx.getTextFile))
		ghx.Router.Add(NewRoute(http.MethodGet, getLooseObject, ghx.getLooseObject))
		ghx.Router.Add(NewRoute(http.MethodGet, getPackFile, ghx.getPackFile))
		ghx.Router.Add(NewRoute(http.MethodGet, getIdxFile, ghx.getIdxFile))
	}
	return ghx, nil
}

type GitHTTPXfer struct {
	Git    *git
	Router *router
	Event  *event
	logger Logger
}

func (ghx *GitHTTPXfer) SetLogger(logger Logger) {
	ghx.logger = logger
}

func (ghx *GitHTTPXfer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	repoPath, filePath, handler, err := ghx.matchRouting(r.Method, r.URL.Path)
	switch err.(type) {
	case *URLNotFoundError:
		RenderNotFound(rw)
		return
	case *MethodNotAllowedError:
		RenderMethodNotAllowed(rw, r)
		return
	}

	ctx := NewContext(rw, r, repoPath, filePath)

	ghx.Event.emit(AfterMatchRouting, ctx)

	if !ghx.Git.Exists(ctx.RepoPath()) {
		RenderNotFound(ctx.Response().Writer)
		return
	}

	handler(ctx)
}

func (ghx *GitHTTPXfer) matchRouting(method, path string) (repoPath string, filePath string, handler HandlerFunc, err error) {
	match, route, err := ghx.Router.Match(method, path)
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
	BeforeUploadPack  EventKey = "before-upload-pack"
	BeforeReceivePack EventKey = "before-receive-pack"
	AfterMatchRouting EventKey = "after-match-routing"
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

func (ghx *GitHTTPXfer) serviceRPCUpload(ctx Context) {
	ghx.Event.emit(BeforeUploadPack, ctx)
	ghx.serviceRPC(ctx, uploadPack)
}

func (ghx *GitHTTPXfer) serviceRPCReceive(ctx Context) {
	ghx.Event.emit(BeforeReceivePack, ctx)
	ghx.serviceRPC(ctx, receivePack)
}

func (ghx *GitHTTPXfer) serviceRPC(ctx Context, rpc string) {

	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	if !ghx.Git.HasAccess(req, rpc, true) {
		RenderNoAccess(res.Writer)
		return
	}

	var body io.ReadCloser
	var err error

	if req.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(req.Body)
		if err != nil {
			ghx.logger.Error("failed to create a reader reading the given reader. ", err.Error())
			RenderInternalServerError(res.Writer)
			return
		}
	} else {
		body = req.Body
	}
	defer body.Close()

	args := []string{rpc, "--stateless-rpc", "."}
	cmd := ghx.Git.GitCommand(repoPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		ghx.logger.Error("failed to get pipe that will be connected to the command's standard input. ", err.Error())
		RenderInternalServerError(res.Writer)
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ghx.logger.Error("failed to get pipe that will be connected to the command's standard output. ", err.Error())
		RenderInternalServerError(res.Writer)
		return
	}
	defer stdout.Close()

	if err = cmd.Start(); err != nil {
		ghx.logger.Error("failed to starts the specified command. ", err.Error())
		RenderInternalServerError(res.Writer)
		return
	}

	if _, err := io.Copy(stdin, body); err != nil {
		ghx.logger.Error("failed to write the request body to standard input. ", err.Error())
		RenderInternalServerError(res.Writer)
		return
	}

	res.SetContentType(fmt.Sprintf("application/x-git-%s-result", rpc))
	res.WriteHeader(http.StatusOK)

	if _, err := io.Copy(res.Writer, stdout); err != nil {
		ghx.logger.Error("failed to write the standard output to response. ", err.Error())
		return
	}


	if err = cmd.Wait(); err != nil {
		ghx.logger.Error("specified command fails to run or doesn't complete successfully. ", err.Error())
	}
}

func (ghx *GitHTTPXfer) getInfoRefs(ctx Context) {
	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	serviceName := getServiceType(req)
	if !ghx.Git.HasAccess(req, serviceName, false) {
		ghx.Git.UpdateServerInfo(repoPath)
		res.HdrNocache()
		if err := ghx.sendFile("text/plain; charset=utf-8", ctx); err != nil {
			RenderNotFound(res.Writer)
			return
		}
	}

	refs, err := ghx.Git.GetInfoRefs(repoPath, serviceName)
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

func (ghx *GitHTTPXfer) getInfoPacks(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ghx.sendFile("text/plain; charset=utf-8", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ghx *GitHTTPXfer) getLooseObject(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ghx.sendFile("application/x-git-loose-object", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ghx *GitHTTPXfer) getPackFile(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ghx.sendFile("application/x-git-packed-objects", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}

}

func (ghx *GitHTTPXfer) getIdxFile(ctx Context) {
	ctx.Response().HdrCacheForever()
	if err := ghx.sendFile("application/x-git-packed-objects-toc", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ghx *GitHTTPXfer) getTextFile(ctx Context) {
	ctx.Response().HdrNocache()
	if err := ghx.sendFile("text/plain", ctx); err != nil {
		RenderNotFound(ctx.Response().Writer)
	}
}

func (ghx *GitHTTPXfer) sendFile(contentType string, ctx Context) error {
	res, req, repoPath, filePath := ctx.Response(), ctx.Request(), ctx.RepoPath(), ctx.FilePath()
	fileInfo, err := ghx.Git.GetRequestFileInfo(repoPath, filePath)
	if err != nil {
		return err
	}
	res.SetContentType(contentType)
	res.SetContentLength(fmt.Sprintf("%d", fileInfo.Size()))
	res.SetLastModified(fileInfo.ModTime().Format(http.TimeFormat))
	http.ServeFile(res.Writer, req, fileInfo.AbsolutePath)
	return nil
}
