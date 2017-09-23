package archivehandler

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

var (
	Pattern = regexp.MustCompile("(.*?)/archive/.*?\\.(zip|tar)$")
	Method  = http.MethodGet
)

func New(ght *githttptransfer.GitHttpTransfer) *archiveHandler {
	return &archiveHandler{ght}
}

type archiveHandler struct {
	*githttptransfer.GitHttpTransfer
}

func (ght *archiveHandler) HandlerFunc(ctx githttptransfer.Context) error {

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
