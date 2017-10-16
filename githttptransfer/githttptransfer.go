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
	serviceRPCUpload  = regexp.MustCompile("(.*?)/git-upload-pack$")
	serviceRPCReceive = regexp.MustCompile("(.*?)/git-receive-pack$")
	getInfoRefs       = regexp.MustCompile("(.*?)/info/refs$")
	getHead           = regexp.MustCompile("(.*?)/HEAD$")
	getAlternates     = regexp.MustCompile("(.*?)/objects/info/alternates$")
	getHTTPAlternates = regexp.MustCompile("(.*?)/objects/info/http-alternates$")
	getInfoPacks      = regexp.MustCompile("(.*?)/objects/info/packs$")
	getInfoFile       = regexp.MustCompile("(.*?)/objects/info/[^/]*$")
	getLooseObject    = regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$")
	getPackFile       = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$")
	getIdxFile        = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$")
)

func New(gitRootPath, gitBinPath string, uploadPack, receivePack, dumbProto bool) (*GitHTTPTransfer, error) {

	if gitRootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		gitRootPath = cwd
	}

	git := newGit(gitRootPath, gitBinPath, uploadPack, receivePack)
	router := newRouter()
	event := newEvent()

	ght := &GitHTTPTransfer{git, router, event}
	ght.Router.Add(NewRoute(http.MethodPost, serviceRPCUpload, ght.serviceRPCUpload))
	ght.Router.Add(NewRoute(http.MethodPost, serviceRPCReceive, ght.serviceRPCReceive))
	ght.Router.Add(NewRoute(http.MethodGet, getInfoRefs, ght.getInfoRefs))

	if dumbProto {
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

func (ght *GitHTTPTransfer) matchRouting(method, path string) (repoPath string, filePath string, handler HandlerFunc, err error) {
	match, route, err := ght.Router.Match(method, path)
	if err == nil {
		repoPath = match[1]
		filePath = strings.Replace(path, repoPath+"/", "", 1)
		handler = route.Handler
	}
	return
}

const (
	uploadPack  string = "upload-pack"
	receivePack string = "receive-pack"
)

type HandlerFunc func(ctx Context) error

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

func (ght *GitHTTPTransfer) serviceRPCUpload(ctx Context) error {
	if err := ght.Event.emit(PrepareServiceRPCUpload, ctx); err != nil {
		return err
	}
	return ght.serviceRPC(ctx, uploadPack)
}

func (ght *GitHTTPTransfer) serviceRPCReceive(ctx Context) error {
	if err := ght.Event.emit(PrepareServiceRPCReceive, ctx); err != nil {
		return err
	}
	return ght.serviceRPC(ctx, receivePack)
}

func (ght *GitHTTPTransfer) serviceRPC(ctx Context, rpc string) error {

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

	err = cmd.Start() // could be merged in one statement
	if err != nil {
		return err
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

	return cmd.Wait()

}

func (ght *GitHTTPTransfer) getInfoRefs(ctx Context) error {
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

func (ght *GitHTTPTransfer) getInfoPacks(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("text/plain; charset=utf-8", ctx)
}

func (ght *GitHTTPTransfer) getLooseObject(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-loose-object", ctx)
}

func (ght *GitHTTPTransfer) getPackFile(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-packed-objects", ctx)
}

func (ght *GitHTTPTransfer) getIdxFile(ctx Context) error {
	ctx.Response().HdrCacheForever()
	return ght.sendFile("application/x-git-packed-objects-toc", ctx)
}

func (ght *GitHTTPTransfer) getTextFile(ctx Context) error {
	ctx.Response().HdrNocache()
	return ght.sendFile("text/plain", ctx)
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
