package githttpxfer

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"testing"
)

type TestParams struct {
	gitRootPath string
	gitBinPath  string
	repoName    string
}

var (
	testParams *TestParams
)

func setupTest(t *testing.T) error {

	gitBinPath, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return err
	}

	testParams = new(TestParams)

	gitRootPath, err := ioutil.TempDir("", "githttpxfer")
	if err != nil {
		t.Logf("get temp directory failed, err: %v", err)
		return err
	}
	os.Chdir(gitRootPath)
	testParams.gitRootPath = gitRootPath
	testParams.gitBinPath = gitBinPath
	testParams.repoName = "test.git"
	err = os.Mkdir(testParams.repoName, os.ModeDir|os.ModePerm)
	if err != nil {
		t.Logf("get temp directory failed, err: %v", err)
		return err
	}

	return nil
}

func teardownTest() {
	os.RemoveAll(testParams.gitRootPath)
}

func Test_GitHTTPXfer_GitHTTPXferOption(t *testing.T) {

	if err := setupTest(t); err != nil {
		return
	}
	defer teardownTest()

	tests := []struct {
		description       string
		url               string
		contentsType      string
		expectedCode      int
		gitHTTPXferOption Option
	}{
		{
			description:       "it should return 403 if upload-pack is off",
			url:               "/test.git/git-upload-pack",
			contentsType:      "application/x-git-upload-pack-request",
			expectedCode:      403,
			gitHTTPXferOption: DisableUploadPack(),
		},
		{
			description:       "it should return 403 if receive-pack is off",
			url:               "/test.git/git-receive-pack",
			contentsType:      "application/x-git-receive-pack-request",
			expectedCode:      403,
			gitHTTPXferOption: DisableReceivePack(),
		},
	}

	for _, tc := range tests {

		t.Log(tc.description)

		ghx, err := New(testParams.gitRootPath, testParams.gitBinPath, tc.gitHTTPXferOption)
		if err != nil {
			t.Errorf("GitHTTPXfer instance could not be created. %s", err.Error())
		}

		ts := httptest.NewServer(ghx)
		if ts == nil {
			t.Log("test server is nil.")
			t.FailNow()
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

func Test_GitHTTPXfer_MatchRouting_should_not_match(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	t.Log("it should not match if http method is different")
	var err error
	ghx, err := New(testParams.gitRootPath, testParams.gitBinPath)
	if err != nil {
		t.Errorf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}
	m := http.MethodGet
	u := &url.URL{
		Path: "/base/foo/git-upload-pack",
	}
	_, _, _, err = ghx.matchRouting(m, u)
	if err == nil {
		t.Error("Allowed.")
		return
	}
	if _, is := err.(*MethodNotAllowedError); !is {
		t.Errorf("error is not MethodNotAllowedError' . result: %s", err.Error())
		return
	}
}

func Test_GitHTTPXfer_MatchRouting_should_match(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	ghx, err := New(testParams.gitRootPath, testParams.gitBinPath)
	if err != nil {
		t.Errorf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}

	tests := []struct {
		description      string
		method           string
		u                *url.URL
		expectedRepoPath string
		expectedFilePath string
	}{
		{
			description:      "it should match git-upload-pack",
			method:           http.MethodPost,
			u:                &url.URL{Path: "/base/foo/git-upload-pack"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "git-upload-pack",
		},
		{
			description:      "it should match get-info-refs",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/info/refs"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "info/refs",
		},
		{
			description:      "it should match get-head",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/HEAD"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "HEAD",
		},
		{
			description:      "it should match get-alternates",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/info/alternates"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/alternates",
		},
		{
			description:      "it should match get-http-alternates",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/info/http-alternates"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/http-alternates",
		},
		{
			description:      "it should match get-info-packs",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/info/packs"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/info/packs",
		},
		{
			description:      "it should match get-loose-object",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/3b/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacccccc",
		},
		{
			description:      "it should match get-pack-file",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.pack",
		},
		{
			description:      "it should match get-idx-file",
			method:           http.MethodGet,
			u:                &url.URL{Path: "/base/foo/objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx"},
			expectedRepoPath: "/base/foo",
			expectedFilePath: "objects/pack/pack-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb.idx",
		},
	}

	for _, tc := range tests {
		t.Log(tc.description)
		repoPath, filePath, _, err := ghx.matchRouting(tc.method, tc.u)
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
