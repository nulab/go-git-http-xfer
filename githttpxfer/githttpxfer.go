package githttpxfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	serviceRPCUpload = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/git-upload-pack")
	}
	serviceRPCReceive = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/git-receive-pack")
	}

	getInfoRefs = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/info/refs")
	}

	getHead = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/HEAD")
	}

	getAlternates = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/alternates")
	}

	getHTTPAlternates = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/http-alternates")
	}

	getInfoPacks = func(u *url.URL) *Match {
		return matchSuffix(u.Path, "/objects/info/packs")
	}

	getInfoFileRegexp = regexp.MustCompile(".*?(/objects/info/[^/]*)$")
	getInfoFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getInfoFileRegexp)
	}

	getLooseObjectRegexp = regexp.MustCompile(".*?(/objects/[0-9a-f]{2}/[0-9a-f]{38})$")
	getLooseObject       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getLooseObjectRegexp)
	}

	getPackFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.pack)$")
	getPackFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getPackFileRegexp)
	}

	getIdxFileRegexp = regexp.MustCompile(".*?(/objects/pack/pack-[0-9a-f]{40}\\.idx)$")
	getIdxFile       = func(u *url.URL) *Match {
		return findStringSubmatch(u.Path, getIdxFileRegexp)
	}
)

type Match struct {
	RepoPath, FilePath string
}

func matchSuffix(path, suffix string) *Match {
	if !strings.HasSuffix(path, suffix) {
		return nil
	}
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &Match{repoPath, filePath}
}

func findStringSubmatch(path string, prefix *regexp.Regexp) *Match {
	m := prefix.FindStringSubmatch(path)
	if m == nil {
		return nil
	}
	suffix := m[1]
	repoPath := strings.Replace(path, suffix, "", 1)
	filePath := strings.Replace(path, repoPath+"/", "", 1)
	return &Match{repoPath, filePath}
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
	repoPath, filePath, handler, err := ghx.matchRouting(r.Method, r.URL)
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

func (ghx *GitHTTPXfer) matchRouting(method string, u *url.URL) (repoPath string, filePath string, handler HandlerFunc, err error) {
	match, route, err := ghx.Router.Match(method, u)

	if err == nil {
		repoPath = match.RepoPath
		filePath = match.FilePath
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
	cmd.Env = ctx.Env()

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

	bufIn := bufPool.Get().([]byte)
	defer bufPool.Put(bufIn)
	if _, err := io.CopyBuffer(stdin, body, bufIn); err != nil {
		ghx.logger.Error("failed to write the request body to standard input. ", err.Error())
		RenderInternalServerError(res.Writer)
		return
	}
	// "git-upload-pack" waits for the remaining input and it hangs,
	// so must close it after completing the copy request body to standard input.
	stdin.Close()

	res.SetContentType(fmt.Sprintf("application/x-git-%s-result", rpc))
	res.WriteHeader(http.StatusOK)

	bufOut := bufPool.Get().([]byte)
	defer bufPool.Put(bufOut)
	if _, err := io.CopyBuffer(res.Writer, stdout, bufOut); err != nil {
		ghx.logger.Error("failed to write the standard output to response. ", err.Error())
		return
	}

	if err = cmd.Wait(); err != nil {
		ghx.logger.Error("specified command fails to run or doesn't complete successfully. ", err.Error())
	}
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

func (ghx *GitHTTPXfer) getInfoRefs(ctx Context) {
	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	serviceName := getServiceType(req)
	if !ghx.Git.HasAccess(req, serviceName, false) {
		args := []string{"update-server-info"}
		cmd := ghx.Git.GitCommand(repoPath, args...)
		cmd.Env = ctx.Env()
		cmd.Output()
		res.HdrNocache()
		if err := ghx.sendFile("text/plain; charset=utf-8", ctx); err != nil {
			RenderNotFound(res.Writer)
		}
		return
	}

	args := []string{serviceName, "--stateless-rpc", "--advertise-refs", "."}
	cmd := ghx.Git.GitCommand(repoPath, args...)
	cmd.Env = ctx.Env()
	refs, err := cmd.Output()
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
