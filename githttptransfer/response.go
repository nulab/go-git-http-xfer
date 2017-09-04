package githttptransfer

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	Writer http.ResponseWriter
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Writer: w}
}

func (r *Response) Header() http.Header {
	return r.Writer.Header()
}

func (r *Response) SetContentType(value string) {
	r.Header().Set("Content-Type", value)
}

func (r *Response) SetContentLength(value string) {
	r.Header().Set("Content-Length", value)
}

func (r *Response) SetLastModified(value string) {
	r.Header().Set("Last-Modified", value)
}

func (r *Response) HdrNocache() {
	r.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	r.Header().Set("Pragma", "no-cache")
	r.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func (r *Response) HdrCacheForever() {
	now := time.Now().Unix()
	expires := now + 31536000
	r.Header().Set("Date", fmt.Sprintf("%d", now))
	r.Header().Set("Expires", fmt.Sprintf("%d", expires))
	r.Header().Set("Cache-Control", "public, max-age=31536000")
}

func (r *Response) WriteHeader(code int) {
	r.Writer.WriteHeader(code)
}

func (r *Response) Write(b []byte) (int, error) {
	return r.Writer.Write(b)
}

func (r *Response) Copy(stdout io.Reader) (int64, error) {
	return io.Copy(r.Writer, stdout)
}

func (r *Response) PktFlush() (int, error) {
	return r.Write([]byte("0000"))
}

func (r *Response) PktWrite(str string) (int, error) {
	s := fmt.Sprintf("%04s", strconv.FormatInt(int64(len(str)+4), 16))
	return r.Write([]byte(s + str))
}
