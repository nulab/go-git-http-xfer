package githttptransfer

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"
	"os"
)


const (
	gitRootPath = "/data/git"
	gitBinPath = "/usr/bin/git"
	repoName = "e2e_test.git"
)

func Test_End_To_End_clone_and_push_and_fetch_and_log(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return
	}

	ght := New(gitRootPath, gitBinPath, true, true)

	ts := httptest.NewServer(ght)
	defer ts.Close()


	absRepoPath := ght.Git.GetAbsolutePath(repoName)
	os.Mkdir(absRepoPath, os.ModeDir)

	initCmd := exec.Command("git", "init", "--bare", "--shared")
	initCmd.Dir = absRepoPath
	testCommand(t, initCmd)


	remoteRepoUrl := ts.URL + "/" + repoName
	tempDir, _ := ioutil.TempDir("", "githttptransfer")

	localDirNameA := "test_a"
	localDirNameB := "test_b"
	pathOfDestLocalDirA := path.Join(tempDir, localDirNameA)
	pathOfDestLocalDirB := path.Join(tempDir, localDirNameB)

	cloneCmdA := exec.Command("git", "clone", remoteRepoUrl, pathOfDestLocalDirA)
	testCommand(t, cloneCmdA)

	cloneCmdB := exec.Command("git", "clone", remoteRepoUrl, pathOfDestLocalDirB)
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

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return
	}

	gsh := New(gitRootPath, gitBinPath, true, true)

	ts := httptest.NewServer(gsh)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/" + repoName + "/info/refs")
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

func Test_it_should_return_200_if_request_to_HEAD(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		log.Println("git is not found. so skip test.")
	}

	gsh := New("/data/git", "/usr/bin/git", true, false)

	ts := httptest.NewServer(gsh)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Get(ts.URL + "/e2e_test.git/HEAD")
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

	log.Printf("response body: %s", string(bodyBytes))

}

func Test_it_should_return_200_if_request_to_loose_objects(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		log.Println("git is not found. so skip test.")
	}

	gsh := New("/data/git", "/usr/bin/git", true, false)

	ts := httptest.NewServer(gsh)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Get(ts.URL + "/e2e_test.git/info/refs")
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
	log.Printf("response body: %s", bodyString)

	pattern := regexp.MustCompile("^([0-9a-f]{2})([0-9a-f]{38})\t")
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match")
		return
	}

	res, err = http.Get(ts.URL + "/e2e_test.git/objects/" + m[1] + "/" + m[2])
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. result: %d", res.StatusCode)
	}

}

func Test_it_should_return_200_if_request_to_info_packs(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		log.Println("git is not found. so skip test.")
	}

	gsh := New("/data/git", "/usr/bin/git", true, false)

	gcCmd := gsh.Git.GitCommand("e2e_test.git", []string{"gc"}...)
	testCommand(t, gcCmd)

	ts := httptest.NewServer(gsh)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Get(ts.URL + "/e2e_test.git/objects/info/packs")
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
	log.Printf("response body: %s", bodyString)

	pattern := regexp.MustCompile("^P\\s(pack-[0-9a-f]{40}\\.pack)")
	m := pattern.FindStringSubmatch(bodyString)
	if m == nil {
		t.Error("not match")
		return
	}

	url := ts.URL + "/e2e_test.git/objects/pack/" + m[1]
	res, err = http.Get(url)
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", url, res.StatusCode)
	}

	res, err = http.Get(strings.Replace(url, ".pack", ".idx", 1))
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("StatusCode is not 200. url: %s, result: %d", url, res.StatusCode)
	}

	res, err = http.Get(ts.URL + "/e2e_test.git/objects/info/http-alternates")
	if err != nil {
		t.Errorf("http.Get: %s", err.Error())
	}

	if res.StatusCode != 404 {
		t.Errorf("StatusCode is not 404. result: %d", res.StatusCode)
	}

}

func testCommand(t *testing.T, cmd *exec.Cmd) {
	out, err := cmd.CombinedOutput()
	t.Logf("%s: %s", cmd.Args, string(out))
	if err != nil {
		t.Errorf("testCommand error: %s", err.Error())
	}
}
