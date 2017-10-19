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
func Test_it_should_return_403_if_upload_pack_is_off(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		log.Println("git is not found. so skip test.")
	}

	ght, err := New("/data/git", "/usr/bin/git", WithoutUploadPack())
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}

	ts := httptest.NewServer(ght)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Post(
		ts.URL+"/test.git/git-upload-pack",
		"application/x-git-upload-pack-request",
		nil,
	)
	if err != nil {
		t.Errorf("http.Post: %s", err.Error())
		return
	}

	if res.StatusCode != 403 {
		t.Errorf("StatusCode is not 403. result: %d", res.StatusCode)
		return
	}

}

func Test_it_should_return_403_if_receive_pack_is_off(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		log.Println("git is not found. so skip test.")
	}

	ght, err := New("/data/git", "/usr/bin/git", WithoutReceivePack())
	if err != nil {
		t.Errorf("GitHTTPTransfer instance could not be created. %s", err.Error())
		return
	}

	ts := httptest.NewServer(ght)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Post(
		ts.URL+"/test.git/git-receive-pack",
		"application/x-git-receive-pack-request",
		nil,
	)
	if err != nil {
		t.Errorf("http.Post: %s", err.Error())
		return
	}

	if res.StatusCode != 403 {
		t.Errorf("StatusCode is not 403. result: %d", res.StatusCode)
		return
	}

}

// TODO Could be converted to a table driven test with the next test
func Test_GitSmartHttp_MatchRouting_should_match_git_upload_pack(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_not_match_if_http_method_is_different(t *testing.T) {
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
func Test_GitSmartHttp_MatchRouting_should_match_get_info_refs(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_head(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_alternates(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_http_alternates(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_info_packs(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_loose_object(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_pack_file(t *testing.T) {
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

func Test_GitSmartHttp_MatchRouting_should_match_get_idx_file(t *testing.T) {
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
