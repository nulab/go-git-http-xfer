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
)

func Test_it_should_be_able_to_clone_and_push_and_fetch_log(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return
	}

	gsh := New("/data/git", "/usr/bin/git", true, true)

	ts := httptest.NewServer(gsh)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	remoteUrl := ts.URL + "/test.git"
	tempDir, _ := ioutil.TempDir("", "gitsmarthttp")

	dirA := "test_a"
	dirB := "test_b"
	destDirA := path.Join(tempDir, dirA)
	destDirB := path.Join(tempDir, dirB)

	cloneCmdA := exec.Command("git", "clone", remoteUrl, destDirA)
	testCommand(t, cloneCmdA)

	cloneCmdB := exec.Command("git", "clone", remoteUrl, destDirB)
	testCommand(t, cloneCmdB)

	setUserNameToGitConfigCmd := exec.Command("git", "config", "--global", "user.name", "yuichi.watanabe")
	setUserNameToGitConfigCmd.Dir = destDirA
	testCommand(t, setUserNameToGitConfigCmd)

	setUserEmailToGitConfigCmd := gsh.Git.GitCommand("git", "config", "--global", "user.email", "yuichi.watanabe.ja@gmail.com")
	setUserEmailToGitConfigCmd.Dir = destDirA
	testCommand(t, setUserEmailToGitConfigCmd)

	touchCmd := exec.Command("touch", "README.txt")
	touchCmd.Dir = destDirA
	testCommand(t, touchCmd)

	addCmd := exec.Command("git", "add", "README.txt")
	addCmd.Dir = destDirA
	testCommand(t, addCmd)

	commitCmd := exec.Command("git", "commit", "-m", "first commit")
	commitCmd.Dir = destDirA
	testCommand(t, commitCmd)

	pushCmd := exec.Command("git", "push", "-u", "origin", "master")
	pushCmd.Dir = destDirA
	testCommand(t, pushCmd)

	fetchCmd := exec.Command("git", "fetch")
	fetchCmd.Dir = destDirB
	testCommand(t, fetchCmd)

	logCmd := exec.Command("git", "log", "--oneline", "origin/master", "-1")
	logCmd.Dir = destDirB
	testCommand(t, logCmd)

}

func Test_it_should_return_200_if_request_to_info_refs(t *testing.T) {

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

	res, err := http.Get(ts.URL + "/test.git/info/refs")
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

	res, err := http.Get(ts.URL + "/test.git/HEAD")
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

	res, err := http.Get(ts.URL + "/test.git/info/refs")
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

	res, err = http.Get(ts.URL + "/test.git/objects/" + m[1] + "/" + m[2])
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

	gcCmd := gsh.Git.GitCommand("test.git", []string{"gc"}...)
	testCommand(t, gcCmd)

	ts := httptest.NewServer(gsh)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	res, err := http.Get(ts.URL + "/test.git/objects/info/packs")
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

	url := ts.URL + "/test.git/objects/pack/" + m[1]
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

	res, err = http.Get(ts.URL + "/test.git/objects/info/http-alternates")
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
