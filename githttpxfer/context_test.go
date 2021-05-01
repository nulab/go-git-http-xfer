package githttpxfer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testContext Context

func setupContextTest(t *testing.T) error {
	request, _ := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	recorder := httptest.NewRecorder()
	testContext = NewContext(recorder, request, "test.git", "")
	return nil
}

func teardownContextTest() {
}

func Test_Context_methods_should_succeed(t *testing.T) {
	err := setupContextTest(t)
	if err != nil {
		t.Errorf("error with setupContextTest: %v", err)
		return
	}
	defer teardownContextTest()

	response := testContext.Response()
	if response == nil {
		t.Error("response should not be nil")
	}

	request := testContext.Request()
	if request.Method == http.MethodGet &&
		request.URL.Path == "/" {
		t.Error("request should be a GET on /")
	}

	testContext.SetRequest(request)
	request = testContext.Request()
	if request.Method == http.MethodGet &&
		request.URL.Path == "/" {
		t.Error("request should be a GET on /")
	}

	testContext.SetRepoPath("/tmp")
	repopath := testContext.RepoPath()
	if repopath != "/tmp" {
		t.Error("repopath should be /tmp")
	}

	testContext.SetFilePath("/tmp/file")
	filepath := testContext.FilePath()
	if filepath != "/tmp/file" {
		t.Error("filepath should be /tmp/file")
	}

	testContext.SetEnv([]string{"key=value"})
	env := testContext.Env()
	if env[0] != "key=value" {
		t.Error("env should be key=value")
	}
}
