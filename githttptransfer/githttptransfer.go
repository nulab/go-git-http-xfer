package githttptransfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
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
	getArchive        = regexp.MustCompile("(.*?)/archive/.*?\\.(zip|tar)$")
)

func New(gitRootPath string, gitBinPath string, uploadPack bool, receivePack bool) *GitHttpTransfer {

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

	gsh := &GitHttpTransfer{git, router}
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
	gsh.AddRoute(NewRoute(http.MethodGet, getArchive, gsh.getArchive))
	return gsh
}

type GitHttpTransfer struct {
	Git    *git
	router *router
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

	if !ght.Git.Exists(repoPath) {
		RenderNotFound(rw)
		return
	}

	ctx := NewContext(rw, r, repoPath, filePath)

	if err := handler(ctx); err != nil {
		if os.IsNotExist(err) {
			RenderNotFound(rw)
			return
		}
		switch err.(type) {
		case *NoAccessError:
			RenderNoAccess(rw)
			return
		}
		RenderInternalServerError(rw)
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

func (ght *GitHttpTransfer) serviceRpcUpload(ctx Context) error {
	return ght.serviceRpc(ctx, uploadPack)
}

func (ght *GitHttpTransfer) serviceRpcReceive(ctx Context) error {
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

func (ght *GitHttpTransfer) getArchive(ctx Context) error {

	res, repoPath, filePath := ctx.Response(), ctx.RepoPath(), ctx.FilePath()

	repoName := strings.Split(path.Base(repoPath), ".")[0]
	fileName := path.Base(filePath)
	tree := fileName[0:strings.LastIndex(fileName, ".")]
	ext := path.Ext(fileName)
	format := strings.Replace(ext, ".", "", 1)

	args := []string{"archive", "--format=" + format, "--prefix=" + repoName + "-" + tree + "/", tree}
	cmd := ght.Git.GitCommand(repoPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("getArchive - cmd.StdoutPipe() error: %s", err.Error())
		return err
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		log.Printf("getArchive - cmd.Start() error: %s", err.Error())
		return err
	}

	res.SetContentType("application/octet-stream")
	res.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	res.Header().Add("Content-Transfer-Encoding", "binary")
	res.Writer.WriteHeader(200)

	if _, err := res.Copy(stdout); err != nil {
		log.Printf("getArchive - res.Copy(stdout) error: %s", err.Error())
		return err
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("getArchive - cmd.Wait() error: %s", err.Error())
		return err
	}
	return nil
}
