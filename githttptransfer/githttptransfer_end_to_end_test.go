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

type EndToEndTest struct {
	gitRootPath string
	gitBinPath string
	repoName string

	ght *GitHttpTransfer
	ts *httptest.Server

	absRepoPath string

	remoteRepoUrl string

	tempDir string
}

var (
	endToEndTest = new(EndToEndTest)
)

func setupEndToEndTest(t *testing.T) error {

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return err
	}

	endToEndTest.gitRootPath = "/data/git"
	endToEndTest.gitBinPath = "/usr/bin/git"
	endToEndTest.repoName = "e2e_test.git"

	endToEndTest.ght = New(endToEndTest.gitRootPath, endToEndTest.gitBinPath, true, true)
	endToEndTest.ts = httptest.NewServer(endToEndTest.ght)

	endToEndTest.absRepoPath = endToEndTest.ght.Git.GetAbsolutePath(endToEndTest.repoName)
	os.Mkdir(endToEndTest.absRepoPath, os.ModeDir)

	initCmd := exec.Command("git", "init", "--bare", "--shared")
	initCmd.Dir = endToEndTest.absRepoPath
	testCommand(t, initCmd)

	endToEndTest.remoteRepoUrl = endToEndTest.ts.URL + "/" + endToEndTest.repoName
	tempDir, _ := ioutil.TempDir("", "githttptransfer")
	endToEndTest.tempDir = tempDir
	return nil
}

func teardownEndToEndTest() {
	endToEndTest.ts.Close()
	endToEndTest = new(EndToEndTest)
}

func Test_End_To_End_clone_and_push_and_fetch_and_log(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	localDirNameA := "test_a"
	localDirNameB := "test_b"
	pathOfDestLocalDirA := path.Join(endToEndTest.tempDir, localDirNameA)
	pathOfDestLocalDirB := path.Join(endToEndTest.tempDir, localDirNameB)

	cloneCmdA := exec.Command("git", "clone", endToEndTest.remoteRepoUrl, pathOfDestLocalDirA)
	testCommand(t, cloneCmdA)

	cloneCmdB := exec.Command("git", "clone", endToEndTest.remoteRepoUrl, pathOfDestLocalDirB)
	testCommand(t, cloneCmdB)

	setUserNameToGitConfigCmd := exec.Command("git", "config", "--global", "user.name", "yuichi.watanabe")
	setUserNameToGitConfigCmd.Dir = pathOfDestLocalDirA
	testCommand(t, setUserNameToGitConfigCmd)

	setUserEmailToGitConfigCmd := exec.Command("git", "config", "--global", "user.email", "yuichi.watanabe.ja@gmail.com")
	setUserEmailToGitConfigCmd.Dir = pathOfDestLocalDirA
	testCommand(t, setUserEmailToGitConfigCmd)

	touchCmd := exec.Command("touch", "README.txt")
	touchCmd.Dir = pathOfDestLocalDirA
	testCommand(t, touchCmd)

	addCmd := exec.Command("git", "add", "README.txt")
	addCmd.Dir = pathOfDestLocalDirA
	testCommand(t, addCmd)

	commitCmd := exec.Command("git", "commit", "-m", "first commit")
	commitCmd.Dir = pathOfDestLocalDirA
	testCommand(t, commitCmd)

	pushCmd := exec.Command("git", "push", "-u", "origin", "master")
	pushCmd.Dir = pathOfDestLocalDirA
	testCommand(t, pushCmd)

	fetchCmd := exec.Command("git", "fetch")
	fetchCmd.Dir = pathOfDestLocalDirB
	testCommand(t, fetchCmd)

	logCmd := exec.Command("git", "log", "--oneline", "origin/master", "-1")
	logCmd.Dir = pathOfDestLocalDirB
	testCommand(t, logCmd)

}

func Test_End_To_End_get_info_refs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTest.remoteRepoUrl + "/info/refs")
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

func Test_End_To_End_get_HEAD(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTest.remoteRepoUrl + "/HEAD")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
	}

	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
	}
}

func Test_End_To_End_loose_objects(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	res, err := http.Get(endToEndTest.remoteRepoUrl + "/info/refs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll error: %s", err.Error())
	}

	bodyString := string(bodyBytes)

	pattern := regexp.MustCompile("^([0-9a-f]{2})([0-9a-f]{38})\t")
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match. body")
		return
	}

	res, err = http.Get(endToEndTest.remoteRepoUrl + "/objects/" + m[1] + "/" + m[2])
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
	}

}

func Test_End_To_End_get_info_packs(t *testing.T) {

	if err := setupEndToEndTest(t); err != nil {
		return
	}
	defer teardownEndToEndTest()

	gcCmd := exec.Command("git", "gc")
	gcCmd.Dir = endToEndTest.absRepoPath
	testCommand(t, gcCmd)

	res, err := http.Get(endToEndTest.remoteRepoUrl + "/objects/info/packs")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d, url: %s", res.StatusCode, res.Request.Host + res.Request.URL.RequestURI())
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

	infoPacksUrl := endToEndTest.remoteRepoUrl + "/objects/pack/" + m[1]
	res, err = http.Get(infoPacksUrl)
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksUrl, res.StatusCode)
	}

	res, err = http.Get(strings.Replace(infoPacksUrl, ".pack", ".idx", 1))
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", infoPacksUrl, res.StatusCode)
	}

	res, err = http.Get(endToEndTest.ts.URL + "/objects/info/http-alternates")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 404 {
		t.Errorf("StatusCode is not 404. result: %d", res.StatusCode)
	}

}

func testCommand(t *testing.T, cmd *exec.Cmd) {
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("testCommand error: %s", err.Error())
	}
}
