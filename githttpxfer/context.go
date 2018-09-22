package githttpxfer

import (
	"net/http"
)

type (
	Context interface {
		Response() *Response
		Request() *http.Request
		SetRequest(r *http.Request)
		RepoPath() string
		SetRepoPath(repoPath string)
		FilePath() string
		SetFilePath(filePath string)
		Env() []string
		SetEnv(env []string)
	}

	context struct {
		response *Response
		request  *http.Request
		repoPath string
		filePath string
		env      []string
	}
)

func NewContext(rw http.ResponseWriter, r *http.Request, repoPath string, filePath string) Context {
	return &context{
		response: NewResponse(rw),
		request:  r,
		repoPath: repoPath,
		filePath: filePath,
	}
}

func (c *context) Response() *Response {
	return c.response
}

func (c *context) Request() *http.Request {
	return c.request
}

func (c *context) SetRequest(r *http.Request) {
	c.request = r
}

func (c *context) RepoPath() string {
	return c.repoPath
}

func (c *context) SetRepoPath(repoPath string) {
	c.repoPath = repoPath
}

func (c *context) FilePath() string {
	return c.filePath
}

func (c *context) SetFilePath(filePath string) {
	c.filePath = filePath
}

func (c *context) Env() []string {
	return c.env
}

func (c *context) SetEnv(env []string) {
	c.env = env
}
