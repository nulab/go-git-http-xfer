package githttpxfer

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"
)

type EndToEndTestParams struct {
	gitRootPath    string
	gitBinPath     string
	repoName       string
	absRepoPath    string
	remoteRepoURL  string
	workingDirPath string // Ex: output destination of git clone.
	head           string
	ghx            *GitHTTPXfer
	ts             *httptest.Server
}

var (
	endToEndTestParams *EndToEndTestParams
)

func setupEndToEndTest(t *testing.T) error {

	gitBinPath, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return err
	}

	endToEndTestParams = new(EndToEndTestParams)

	gitRootPath, err := ioutil.TempDir("", "githttpxfer")
	if err != nil {
		t.Logf("get temp directory failed, err: %v", err)
		return err
	}
	os.Chdir(gitRootPath)
	endToEndTestParams.gitRootPath = gitRootPath
	endToEndTestParams.gitBinPath = gitBinPath
	endToEndTestParams.repoName = "e2e_test.git"
	endToEndTestParams.head = "master"

	ghx, err := New(endToEndTestParams.gitRootPath, endToEndTestParams.gitBinPath)
	if err != nil {
		t.Errorf("GitHTTPXfer instance could not be created. %s", err.Error())
		return err
	}

	endToEndTestParams.ghx = ghx
	endToEndTestParams.ghx.Event.On(BeforeUploadPack, func(ctx Context) {
		t.Log("prepare run service rpc upload.")
	})
	endToEndTestParams.ghx.Event.On(BeforeReceivePack, func(ctx Context) {
		t.Log("prepare run service rpc receive.")
	})
	endToEndTestParams.ghx.Event.On(AfterMatchRouting, func(ctx Context) {
		t.Log("after match routing.")
	})

	endToEndTestParams.ts = httptest.NewServer(endToEndTestParams.ghx)

	endToEndTestParams.absRepoPath = endToEndTestParams.ghx.Git.GetAbsolutePath(endToEndTestParams.repoName)
	os.Mkdir(endToEndTestParams.absRepoPath, os.ModeDir|os.ModePerm)

	if _, err := execCmd(endToEndTestParams.absRepoPath, "git", "init", "--bare", "--shared"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return err
	}

	endToEndTestParams.remoteRepoURL = endToEndTestParams.ts.URL + "/" + endToEndTestParams.repoName
	tempDir, _ := ioutil.TempDir("", "githttpxfer")
	endToEndTestParams.workingDirPath = tempDir
	return nil
}

func teardownEndToEndTest() {
	endToEndTestParams.ts.Close()
	os.RemoveAll(endToEndTestParams.gitRootPath)
}

func execCmd(dir string, name string, arg ...string) ([]byte, error) {
	c := exec.Command(name, arg...)
	c.Dir = dir
	return c.CombinedOutput()
}

func Test_End_To_End_it_should_succeed_clone_and_push_and_fetch_and_log(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	remoteRepoUrl := endToEndTestParams.remoteRepoURL

	destDirNameA := "test_a"
	destDirNameB := "test_b"
	destDirNameC := "test_c"
	destDirPathA := path.Join(endToEndTestParams.workingDirPath, destDirNameA)
	destDirPathB := path.Join(endToEndTestParams.workingDirPath, destDirNameB)
	destDirPathC := path.Join(endToEndTestParams.workingDirPath, destDirNameC)

	if _, err := execCmd("", "git", "clone", remoteRepoUrl, destDirPathA); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd("", "git", "clone", remoteRepoUrl, destDirPathB); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathA, "git", "config", "--global", "user.name", "John Smith"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathA, "git", "config", "--global", "user.email", "js@example.com"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathA, "touch", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathA, "git", "add", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathA, "git", "commit", "-m", "first commit"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	output, err := ioutil.ReadFile(path.Join(destDirPathA, ".git/HEAD"))
	if err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	endToEndTestParams.head = strings.Join(strings.Split(strings.TrimSuffix(string(output), "\n"), "/")[2:], "/")
	if endToEndTestParams.head == "" {
		t.Error("could not figure out HEAD")
		return
	}
	if _, err := execCmd(destDirPathA, "git", "push", "-u", "origin", endToEndTestParams.head); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathB, "git", "fetch"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathB, "git", "log", "--oneline", "origin/"+endToEndTestParams.head, "-1"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd("", "git", "clone", "--depth=1", remoteRepoUrl, destDirPathC); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

}

func Test_End_To_End_it_should_succeed_request_to_get_info_refs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTestParams.remoteRepoURL + "/info/refs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
		return
	}

	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
		return
	}

}

func Test_End_To_End_it_should_succeed_request_to_get_HEAD(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTestParams.remoteRepoURL + "/HEAD")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
		return
	}

	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
		return
	}
}

func Test_End_To_End_it_should_succeed_request_to_loose_objects(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	destDirName := "test"
	destDirPath := path.Join(endToEndTestParams.workingDirPath, destDirName)

	if _, err := execCmd("", "git", "clone", endToEndTestParams.remoteRepoURL, destDirPath); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "config", "--global", "user.name", "John Smith"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "config", "--global", "user.email", "js@example.com"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "touch", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "add", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "commit", "-m", "first commit"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	output, err := ioutil.ReadFile(path.Join(destDirPath, ".git/HEAD"))
	if err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	endToEndTestParams.head = strings.Join(strings.Split(strings.TrimSuffix(string(output), "\n"), "/")[2:], "/")
	if endToEndTestParams.head == "" {
		t.Error("could not figure out HEAD")
		return
	}
	if _, err := execCmd(destDirPath, "git", "push", "-u", "origin", endToEndTestParams.head); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	res, err := http.Get(endToEndTestParams.remoteRepoURL + "/info/refs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
		return
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
		return
	}

	bodyString := string(bodyBytes)

	pattern := regexp.MustCompile("^([0-9a-f]{2})([0-9a-f]{38})\t")
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match. body")
		return
	}

	res, err = http.Get(endToEndTestParams.remoteRepoURL + "/objects/" + m[1] + "/" + m[2])
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
		return
	}

}

// TODO Should succeed but check if not 404. Rename the test
func Test_End_To_End_it_should_succeed_request_to_get_info_packs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	destDirName := "test"
	destDirPath := path.Join(endToEndTestParams.workingDirPath, destDirName)

	if _, err := execCmd("", "git", "clone", endToEndTestParams.remoteRepoURL, destDirPath); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "config", "--global", "user.name", "John Smith"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "config", "--global", "user.email", "js@example.com"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "touch", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "add", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPath, "git", "commit", "-m", "first commit"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	output, err := ioutil.ReadFile(path.Join(destDirPath, ".git/HEAD"))
	if err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	endToEndTestParams.head = strings.Join(strings.Split(strings.TrimSuffix(string(output), "\n"), "/")[2:], "/")
	if endToEndTestParams.head == "" {
		t.Error("could not figure out HEAD")
		return
	}
	if _, err := execCmd(destDirPath, "git", "push", "-u", "origin", endToEndTestParams.head); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(endToEndTestParams.absRepoPath, "git", "gc"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	res, err := http.Get(endToEndTestParams.remoteRepoURL + "/objects/info/packs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != http.StatusOK {
		url := res.Request.Host + res.Request.URL.RequestURI()
		t.Errorf("StatusCode is not 200. result: %d, url: %s", res.StatusCode, url)
		return
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
		return
	}

	bodyString := string(bodyBytes)

	pattern := regexp.MustCompile(`^P\s(pack-[0-9a-f]{40}\.pack)`)
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match")
		return
	}

	infoPacksURL := endToEndTestParams.remoteRepoURL + "/objects/pack/" + m[1]
	res, err = http.Get(infoPacksURL)
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksURL, res.StatusCode)
		return
	}

	res, err = http.Get(strings.Replace(infoPacksURL, ".pack", ".idx", 1))
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksURL, res.StatusCode)
		return
	}

	res, err = http.Get(endToEndTestParams.ts.URL + "/objects/info/http-alternates")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode is not 404. result: %d", res.StatusCode)
		return
	}
}
