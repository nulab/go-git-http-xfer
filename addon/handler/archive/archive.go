package archive

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/nulab/go-git-http-xfer/githttpxfer"
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

func New(ghx *githttpxfer.GitHTTPXfer) *gitHTTPXfer {
	return &gitHTTPXfer{ghx}
}

type gitHTTPXfer struct {
	*githttpxfer.GitHTTPXfer
}

func (ghx *gitHTTPXfer) Archive(ctx githttpxfer.Context) {

	res, repoPath, filePath := ctx.Response(), ctx.RepoPath(), ctx.FilePath()

	repoName := strings.Split(path.Base(repoPath), ".")[0]
	fileName := path.Base(filePath)
	tree := fileName[0:strings.LastIndex(fileName, ".")]
	ext := path.Ext(fileName)
	format := strings.Replace(ext, ".", "", 1)

	args := []string{"archive", "--format=" + format, "--prefix=" + repoName + "-" + tree + "/", tree}
	cmd := ghx.Git.GitCommand(repoPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		githttpxfer.RenderInternalServerError(res.Writer)
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		githttpxfer.RenderInternalServerError(res.Writer)
		return
	}

	res.SetContentType("application/octet-stream")
	res.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	res.Header().Add("Content-Transfer-Encoding", "binary")

	if _, err := res.Copy(stdout); err != nil {
		githttpxfer.RenderInternalServerError(res.Writer)
		return
	}
	if err := cmd.Wait(); err != nil {
		githttpxfer.RenderInternalServerError(res.Writer)
	}
}
