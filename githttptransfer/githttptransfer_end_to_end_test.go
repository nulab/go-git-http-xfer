package githttptransfer

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path"
	"regexp"
	"testing"
	"os"
	"strings"
)

type EndToEndTestParams struct {
	gitRootPath    string
	gitBinPath     string
	repoName       string
	ght            *GitHttpTransfer
	ts             *httptest.Server
	absRepoPath    string
	remoteRepoUrl  string
	workingDirPath string // Ex: output destination of git clone.
}

var (
	endToEndTestParams *EndToEndTestParams
)

func setupEndToEndTest(t *testing.T) error {

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return err
	}

	endToEndTestParams = new(EndToEndTestParams)

	endToEndTestParams.gitRootPath = "/data/git"
	endToEndTestParams.gitBinPath = "/usr/bin/git"
	endToEndTestParams.repoName = "e2e_test.git"

	endToEndTestParams.ght = New(endToEndTestParams.gitRootPath, endToEndTestParams.gitBinPath, true, true)
	endToEndTestParams.ts = httptest.NewServer(endToEndTestParams.ght)

	endToEndTestParams.absRepoPath = endToEndTestParams.ght.Git.GetAbsolutePath(endToEndTestParams.repoName)
	os.Mkdir(endToEndTestParams.absRepoPath, os.ModeDir)


	if _, err := execCmd(endToEndTestParams.absRepoPath, "git", "init", "--bare", "--shared"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return err
	}

	endToEndTestParams.remoteRepoUrl = endToEndTestParams.ts.URL + "/" + endToEndTestParams.repoName
	tempDir, _ := ioutil.TempDir("", "githttptransfer")
	endToEndTestParams.workingDirPath = tempDir
	return nil
}

func teardownEndToEndTest() {
	endToEndTestParams.ts.Close()
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

	remoteRepoUrl := endToEndTestParams.remoteRepoUrl

	destDirNameA := "test_a"
	destDirNameB := "test_b"
	destDirPathA := path.Join(endToEndTestParams.workingDirPath, destDirNameA)
	destDirPathB := path.Join(endToEndTestParams.workingDirPath, destDirNameB)

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

	if _, err := execCmd(destDirPathA, "git", "push", "-u", "origin", "master"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathB, "git", "fetch"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDirPathB, "git", "log", "--oneline", "origin/master", "-1"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

}

func Test_End_To_End_it_should_succeed_request_to_get_info_refs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTestParams.remoteRepoUrl + "/info/refs")
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

	res, err := http.Get(endToEndTestParams.remoteRepoUrl + "/HEAD")
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

	res, err := http.Get(endToEndTestParams.remoteRepoUrl + "/info/refs")
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

	res, err = http.Get(endToEndTestParams.remoteRepoUrl + "/objects/" + m[1] + "/" + m[2])
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
		return
	}

}

func Test_End_To_End_it_should_succeed_request_to_get_info_packs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	if _, err := execCmd(endToEndTestParams.absRepoPath, "git", "gc"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	res, err := http.Get(endToEndTestParams.remoteRepoUrl + "/objects/info/packs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
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

	pattern := regexp.MustCompile("^P\\s(pack-[0-9a-f]{40}\\.pack)")
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match")
		return
	}

	infoPacksUrl := endToEndTestParams.remoteRepoUrl + "/objects/pack/" + m[1]
	res, err = http.Get(infoPacksUrl)
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksUrl, res.StatusCode)
		return
	}

	res, err = http.Get(strings.Replace(infoPacksUrl, ".pack", ".idx", 1))
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksUrl, res.StatusCode)
		return
	}

	res, err = http.Get(endToEndTestParams.ts.URL + "/objects/info/http-alternates")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
		return
	}

	if res.StatusCode != 404 {
		t.Errorf("StatusCode is not 404. result: %d", res.StatusCode)
		return
	}

}

