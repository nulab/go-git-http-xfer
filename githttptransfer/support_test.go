package githttptransfer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Can be converted to a table driven test
// https://github.com/golang/go/wiki/TableDrivenTests
func Test_GetServiceType_should_return_git_upload_pack(t *testing.T) {
	m := http.MethodPost
	p := "http://example.com/base/foo/git-upload-pack?service=git-upload-pack"
	r := httptest.NewRequest(m, p, nil)

	expected := "upload-pack"

	if serviceType := getServiceType(r); expected != serviceType {
		t.Errorf("service type is not %s . result: %s", expected, serviceType)
	}
}

func Test_GetServiceType_should_return_git_receive_pack(t *testing.T) {
	m := http.MethodPost
	p := "http://example.com/base/foo/git-receive-pack?service=git-receive-pack"
	r := httptest.NewRequest(m, p, nil)

	expected := "receive-pack"

	if serviceType := getServiceType(r); expected != serviceType {
		t.Errorf("service type is not %s . result: %s", expected, serviceType)
	}
}

func Test_GetServiceType_should_return_empty(t *testing.T) {
	m := http.MethodPost
	p := "http://example.com/base/foo/git-upload-pack?service=foo-upload-pack"
	r := httptest.NewRequest(m, p, nil)

	expected := ""

	if serviceType := getServiceType(r); expected != serviceType {
		t.Errorf("service type is not %s . result: %s", "empty", serviceType)
	}
}
