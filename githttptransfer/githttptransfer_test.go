package githttptransfer

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
)

func Test_GitHTTPTransfer_GitHTTPTransferOption(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		log.Println("git is not found. so skip test.")
	}

	tests := []struct {
		description           string
		url                   string
		contentsType          string
		expectedCode          int
		gitHTTPTransferOption Option
	}{
		{
			description:           "it should return 403 if upload-pack is off",
			url:                   "/test.git/git-upload-pack",
			contentsType:          "application/x-git-upload-pack-request",
			expectedCode:          403,
			gitHTTPTransferOption: WithoutUploadPack(),
		},
		{
			description:           "it should return 403 if receive-pack is off",
			url:                   "/test.git/git-receive-pack",
			contentsType:          "application/x-git-receive-pack-request",
			expectedCode:          403,
			gitHTTPTransferOption: WithoutReceivePack(),
		},
	}

	for _, tc := range tests {

		t.Log(tc.description)

		ght, err := New("/data/git", "/usr/bin/git", tc.gitHTTPTransferOption)
		if err != nil {
			t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		}

		ts := httptest.NewServer(ght)
		if ts == nil {
			t.Error("test server is nil.")
		}

		res, err := http.Post(
			ts.URL+tc.url,
			tc.contentsType,
			nil,
		)
		if err != nil {
			t.Errorf("http.Post: %s", err.Error())
		}

		if res.StatusCode != tc.expectedCode {
			t.Errorf("StatusCode is not %d. result: %d", tc.expectedCode, res.StatusCode)
		}
		ts.Close()
	}

}

func Test_GitHTTPTransfer_MatchRouting_should_not_match(t *testing.T) {
	t.Log("it should not match if http method is different")
	var err error
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/git-upload-pack"
	_, _, _, err = ght.matchRouting(m, p)
	if err == nil {
		t.Error("Allowed.")
		return
	}
	if _, is := err.(*MethodNotAllowedError); !is {
		t.Errorf("error is not MethodNotAllowedError' . result: %s", err.Error())
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}

	tests := []struct {
		description      string
		method           string
		path             string
		expectedRepoPath string
		expectedFilePath string
	}{
		{
			description:      "it should match git-upload-pack",
			method:           http.MethodPost,
			path:             "/base/foo/git-upload-pack",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "git-upload-pack",
		},
		{
			description:      "it should match get-info-refs",
			method:           http.MethodGet,
			path:             "/base/foo/info/refs",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "info/refs",
		},
		{
			description:      "it should match get-head",
			method:           http.MethodGet,
			path:             "/base/foo/HEAD",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "HEAD",
		},
		{
			description:      "it should match get-alternates",
			method:           http.MethodGet,
			path:             "/base/foo/objects/info/alternates",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/alternates",
		},
		{
			description:      "it should match get-http-alternates",
			method:           http.MethodGet,
			path:             "/base/foo/objects/info/http-alternates",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/http-alternates",
		},
		{
			description:      "it should match get-info-packs",
			method:           http.MethodGet,
			path:             "/base/foo/objects/info/packs",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/packs",
		},
		{
			description:      "it should match get-loose-object",
			method:           http.MethodGet,
			path:             "/base/foo/objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc",
		},
		{
			description:      "it should match get-pack-file",
			method:           http.MethodGet,
			path:             "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack",
		},
		{
			description:      "it should match get-idx-file",
			method:           http.MethodGet,
			path:             "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx",
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx",
		},
	}

	for _, tc := range tests {
		t.Log(tc.description)
		repoPath, filePath, _, err := ght.matchRouting(tc.method, tc.path)
		if err != nil {
			t.Errorf("error is %s", err.Error())
			return
		}
		if repoPath != tc.expectedRepoPath {
			t.Errorf("repository path is not %s . result: %s", tc.expectedRepoPath, repoPath)
			return
		}
		if filePath != tc.expectedFilePath {
			t.Errorf("file path is not %s . result: %s", tc.expectedFilePath, filePath)
			return
		}
	}
}
