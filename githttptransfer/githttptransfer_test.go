package githttptransfer

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
)

// TODO Could be converted to a table driven test with the next test
// https://github.com/golang/go/wiki/TableDrivenTests
func Test_GitHTTPTransfer_GitHTTPTransferOption(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		log.Println("git is not found. so skip test.")
	}

	tests := []struct {
		description  string
		url          string
		contentsType string
		expectedCode int
		gitHTTPTransferOption GitHTTPTransferOption
	}{
		{
			description:  "it should return 403 if upload-pack is off",
			url:          "/test.git/git-upload-pack",
			contentsType: "application/x-git-upload-pack-request",
			expectedCode: 403,
			gitHTTPTransferOption: WithoutUploadPack(),
		},
		{
			description:  "it should return 403 if receive-pack is off",
			url:          "/test.git/git-receive-pack",
			contentsType: "application/x-git-receive-pack-request",
			expectedCode: 403,
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

// TODO Could be converted to a table driven test with the next test
func Test_GitHTTPTransfer_MatchRouting_should_match_git_upload_pack(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodPost
	p := "/base/foo/git-upload-pack"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "git-upload-pack"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_not_match_if_http_method_is_different(t *testing.T) {
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

// TODO Could be converted to a table driven test
// https://github.com/golang/go/wiki/TableDrivenTests
func Test_GitHTTPTransfer_MatchRouting_should_match_get_info_refs(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/info/refs"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "info/refs"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_head(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/HEAD"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "HEAD"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_alternates(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/info/alternates"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/info/alternates"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_http_alternates(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/info/http-alternates"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/info/http-alternates"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_info_packs(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/info/packs"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/info/packs"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_loose_object(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_pack_file(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}

func Test_GitHTTPTransfer_MatchRouting_should_match_get_idx_file(t *testing.T) {
	ght, err := New("", "/usr/bin/git")
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	p := "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx"
	expectedRepoPath := "/base/foo"
	expectedFilePath := "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx"
	repoPath, filePath, _, err := ght.matchRouting(m, p)
	if err != nil {
		t.Errorf("error is %s", err.Error())
		return
	}
	if repoPath != expectedRepoPath {
		t.Errorf("repository path is not %s . result: %s", expectedRepoPath, repoPath)
		return
	}
	if filePath != expectedFilePath {
		t.Errorf("file path is not %s . result: %s", expectedFilePath, filePath)
		return
	}
}
