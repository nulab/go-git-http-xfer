package githttptransfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	serviceRpcUpload  = regexp.MustCompile("(.*?)/git-upload-pack$")
	serviceRpcReceive = regexp.MustCompile("(.*?)/git-receive-pack$")
	getInfoRefs       = regexp.MustCompile("(.*?)/info/refs$")
	getHead           = regexp.MustCompile("(.*?)/HEAD$")
	getAlternates     = regexp.MustCompile("(.*?)/objects/info/alternates$")
	getHttpAlternates = regexp.MustCompile("(.*?)/objects/info/http-alternates$")
	getInfoPacks      = regexp.MustCompile("(.*?)/objects/info/packs$")
	getInfoFile       = regexp.MustCompile("(.*?)/objects/info/[^/]*$")
	getLooseObject    = regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$")
	getPackFile       = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$")
	getIdxFile        = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$")
)

func New(gitRootPath, gitBinPath string, uploadPack, receivePack bool) *GitHttpTransfer {

	if gitRootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Invalid GitRootPath. os.Getwd() error: %s", err.Error())
			return nil
		}
		gitRootPath = cwd
	}

	git := newGit(gitRootPath, gitBinPath, uploadPack, receivePack)
	router := newRouter()
	event := newEvent()

	gsh := &GitHttpTransfer{git, router, event}
	gsh.AddRoute(NewRoute(http.MethodPost, serviceRpcUpload, gsh.serviceRpcUpload))
	gsh.AddRoute(NewRoute(http.MethodPost, serviceRpcReceive, gsh.serviceRpcReceive))
	gsh.AddRoute(NewRoute(http.MethodGet, getInfoRefs, gsh.getInfoRefs))
	gsh.AddRoute(NewRoute(http.MethodGet, getHead, gsh.getTextFile))
	gsh.AddRoute(NewRoute(http.MethodGet, getAlternates, gsh.getTextFile))
	gsh.AddRoute(NewRoute(http.MethodGet, getHttpAlternates, gsh.getTextFile))
	gsh.AddRoute(NewRoute(http.MethodGet, getInfoPacks, gsh.getInfoPacks))
	gsh.AddRoute(NewRoute(http.MethodGet, getInfoFile, gsh.getTextFile))
	gsh.AddRoute(NewRoute(http.MethodGet, getLooseObject, gsh.getLooseObject))
	gsh.AddRoute(NewRoute(http.MethodGet, getPackFile, gsh.getPackFile))
	gsh.AddRoute(NewRoute(http.MethodGet, getIdxFile, gsh.getIdxFile))
	return gsh
}

type GitHttpTransfer struct {
	Git    *git
	router *router
	Event  Event
}

func (ght *GitHttpTransfer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	repoPath, filePath, handler, err := ght.MatchRouting(r.Method, r.URL.Path)
	switch err.(type) {
	case *UrlNotFoundError:
		RenderNotFound(rw)
		return
	case *MethodNotAllowedError:
		RenderMethodNotAllowed(rw, r)
		return
	}

	ctx := NewContext(rw, r, repoPath, filePath)

	if err := ght.Event.emit(AfterMatchRouting, ctx); err != nil {
		RenderInternalServerError(ctx.Response().Writer)
		return
	}

	if !ght.Git.Exists(ctx.RepoPath()) {
		RenderNotFound(ctx.Response().Writer)
		return
	}

	if err := handler(ctx); err != nil {
		if os.IsNotExist(err) {
			RenderNotFound(ctx.Response().Writer)
			return
		}
		switch err.(type) {
		case *NoAccessError:
			RenderNoAccess(ctx.Response().Writer)
			return
		}
		RenderInternalServerError(ctx.Response().Writer)
	}
}

func (ght *GitHttpTransfer) MatchRouting(method, path string) (repoPath string, filePath string, handler HandlerFunc, err error) {
	match, route, err := ght.router.match(method, path)
	if err == nil {
		repoPath = match[1]
		filePath = strings.Replace(path, repoPath+"/", "", 1)
		handler = route.Handler
	}
	return
}

func (ght *GitHttpTransfer) AddRoute(route *Route) {
	ght.router.add(route)
}

const (
	uploadPack  string = "upload-pack"
	receivePack string = "receive-pack"
)

type HandlerFunc func(ctx Context) error

func newEvent() Event {
	return &event{map[EventKey]HandlerFunc{}}
}

type EventKey string

const (
	PrepareServiceRpcUpload  EventKey = "prepare-service-rpc-upload"
	PrepareServiceRpcReceive EventKey = "prepare-service-rpc-receive"
	AfterMatchRouting        EventKey = "after-match-routing"
)

type Event interface {
	emit(evt EventKey, ctx Context) error
	On(evt EventKey, listener HandlerFunc)
}

type event struct {
	listeners map[EventKey]HandlerFunc
}

func (e *event) emit(evt EventKey, ctx Context) error {
	v, ok := e.listeners[evt]
	if ok {
		return v(ctx)
	}
	return nil
}

func (e *event) On(evt EventKey, listener HandlerFunc) {
	e.listeners[evt] = listener
}

func (ght *GitHttpTransfer) serviceRpcUpload(ctx Context) error {
	if err := ght.Event.emit(PrepareServiceRpcUpload, ctx); err != nil {
		return err
	}
	return ght.serviceRpc(ctx, uploadPack)
}

func (ght *GitHttpTransfer) serviceRpcReceive(ctx Context) error {
	if err := ght.Event.emit(PrepareServiceRpcReceive, ctx); err != nil {
		return err
	}
	return ght.serviceRpc(ctx, receivePack)
}

func (ght *GitHttpTransfer) serviceRpc(ctx Context, rpc string) error {

	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	if !ght.Git.HasAccess(req, rpc, true) {
		return &NoAccessError{Dir: ght.Git.GetAbsolutePath(repoPath)}
	}

	var body io.ReadCloser
	var err error

	if req.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(req.Body)
		if err != nil {
			return err
		}
	} else {
		body = req.Body
	}
	defer body.Close()

	input, _ := ioutil.ReadAll(body)

	res.SetContentType(fmt.Sprintf("application/x-git-%s-result", rpc))

	args := []string{rpc, "--stateless-rpc", "."}
	cmd := ght.Git.GitCommand(repoPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	stdin.Write(input)
	stdin.Close()
	res.Copy(stdout)
	cmd.Wait()

	return nil

}

func (ght *GitHttpTransfer) getInfoRefs(ctx Context) error {
	res, req, repoPath := ctx.Response(), ctx.Request(), ctx.RepoPath()

	serviceName := getServiceType(req)
	if !ght.Git.HasAccess(req, serviceName, false) {
		ght.Git.UpdateServerInfo(repoPath)
		res.HdrNocache()
		return ght.sendFile("text/plain; charset=utf-8", ctx)
	}

	refs, err := ght.Git.GetInfoRefs(repoPath, serviceName)
	if err != nil {
		return err
	}

	res.HdrNocache()
	res.SetContentType(fmt.Sprintf("application/x-git-%s-advertisement", serviceName))
	res.WriteHeader(http.StatusOK)
	res.PktWrite("# service=git-" + serviceName + "\n")
	res.PktFlush()
	res.Write(refs)

	return nil
}

func (ght *GitHttpTransfer) getInfoPacks(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("text/plain; charset=utf-8", ctx)
}

func (ght *GitHttpTransfer) getLooseObject(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-loose-object", ctx)
}

func (ght *GitHttpTransfer) getPackFile(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-packed-objects", ctx)
}

func (ght *GitHttpTransfer) getIdxFile(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-packed-objects-toc", ctx)
}

func (ght *GitHttpTransfer) getTextFile(ctx Context) error {
	ctx.Response().HdrNocache()
	return ght.sendFile("text/plain", ctx)
}

func (ght *GitHttpTransfer) sendFile(contentType string, ctx Context) error {
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
