package archive

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

var (
	archiveRegexp = regexp.MustCompile(".*?(/archive/.*?\\.(zip|tar))$")
	Pattern       = func(path string) (match string) {
		if m := archiveRegexp.FindStringSubmatch(path); m != nil {
			match = m[1]
		}
		return
	}
	Method = http.MethodGet
)

func New(ght *githttptransfer.GitHTTPTransfer) *gitHTTPTransfer {
	return &gitHTTPTransfer{ght}
}

type gitHTTPTransfer struct {
	*githttptransfer.GitHTTPTransfer
}

func (ght *gitHTTPTransfer) Archive(ctx githttptransfer.Context) {

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
		githttptransfer.RenderInternalServerError(res.Writer)
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		githttptransfer.RenderInternalServerError(res.Writer)
		return
	}

	res.SetContentType("application/octet-stream")
	res.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	res.Header().Add("Content-Transfer-Encoding", "binary")

	if _, err := res.Copy(stdout); err != nil {
		githttptransfer.RenderInternalServerError(res.Writer)
		return
	}
	if err := cmd.Wait(); err != nil {
		githttptransfer.RenderInternalServerError(res.Writer)
	}
}
