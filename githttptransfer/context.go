package githttptransfer

import (
	"net/http"
)

type (
	Context interface {
		Response() *Response
		Request() *http.Request
		RepoPath() string
		FilePath() string
	}

	context struct {
		response *Response
		request  *http.Request
		repoPath string
		filePath string
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

func (c *context) RepoPath() string {
	return c.repoPath
}

func (c *context) FilePath() string {
	return c.filePath
}
