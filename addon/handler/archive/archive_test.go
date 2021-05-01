package archive

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/nulab/go-git-http-xfer/githttpxfer"
)

type ArchiveParams struct {
	gitRootPath string
	gitBinPath  string
	repoName    string
	head        string
}

var (
	archiveParams *ArchiveParams
)

func setupArchiveTest(t *testing.T) error {

	gitBinPath, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git e2e test.")
		return err
	}

	archiveParams = new(ArchiveParams)

	gitRootPath, err := ioutil.TempDir("", "githttpxfer")
	if err != nil {
		t.Logf("get temp directory failed, err: %v", err)
		return err
	}
	os.Chdir(gitRootPath)
	archiveParams.gitRootPath = gitRootPath
	archiveParams.gitBinPath = gitBinPath
	archiveParams.repoName = "test.git"
	archiveParams.head = "master"

	return nil
}

func teardownArchiveTest() {
	os.RemoveAll(archiveParams.gitRootPath)
}

func Test_it_should_download_archive_repository(t *testing.T) {

	if err := setupArchiveTest(t); err != nil {
		return
	}
	defer teardownArchiveTest()

	ghx, err := githttpxfer.New(archiveParams.gitRootPath, archiveParams.gitBinPath)
	if err != nil {
		t.Errorf("An instance could not be created. %s", err.Error())
		return
	}

	ghx.Router.Add(githttpxfer.NewRoute(
		Method,
		Pattern,
		New(ghx).Archive,
	))

	ts := httptest.NewServer(ghx)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	repoName := "archive_test.git"
	absRepoPath := ghx.Git.GetAbsolutePath(repoName)
	os.Mkdir(absRepoPath, os.ModeDir|os.ModePerm)

	if _, err := execCmd(absRepoPath, "git", "init", "--bare", "--shared"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	remoteRepoUrl := ts.URL + "/" + repoName

	tempDir, _ := ioutil.TempDir("", "git-http-xfer")
	dir := "archive_test"
	destDir := path.Join(tempDir, dir)

	if _, err := execCmd("", "git", "clone", remoteRepoUrl, destDir); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "git", "config", "--global", "user.name", "John Smith"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "git", "config", "--global", "user.email", "js@example.com"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "touch", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "git", "add", "README.txt"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "git", "commit", "-m", "first commit"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	output, err := ioutil.ReadFile(path.Join(destDir, ".git/HEAD"))
	if err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	archiveParams.head = strings.Join(strings.Split(strings.TrimSuffix(string(output), "\n"), "/")[2:], "/")
	if archiveParams.head == "" {
		t.Error("could not figure out HEAD")
		return
	}

	if _, err := execCmd(destDir, "git", "push", "-u", "origin", archiveParams.head); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "wget", "-O-", remoteRepoUrl+"/archive/"+archiveParams.head+".zip"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "wget", "-O-", remoteRepoUrl+"/archive/"+archiveParams.head+".tar"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

}

func Test_it_should_fail_404_download_archive(t *testing.T) {

	if err := setupArchiveTest(t); err != nil {
		return
	}
	defer teardownArchiveTest()

	ghx, err := githttpxfer.New(archiveParams.gitRootPath, archiveParams.gitBinPath)
	if err != nil {
		t.Errorf("An instance could not be created. %s", err.Error())
		return
	}

	ghx.Router.Add(githttpxfer.NewRoute(
		Method,
		Pattern,
		New(ghx).Archive,
	))

	ts := httptest.NewServer(ghx)
	if ts == nil {
		t.Log("test server is nil.")
		t.FailNow()
	}
	defer ts.Close()

	repoName := "archive_test.git"
	remoteRepoUrl := ts.URL + "/" + repoName

	client := ts.Client()
	request, _ := http.NewRequest(http.MethodGet, remoteRepoUrl, nil)
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("Should not return %v", err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Errorf(
			"Status Code should be 404, current: %d",
			response.StatusCode,
		)
	}
}

func execCmd(dir string, name string, arg ...string) ([]byte, error) {
	c := exec.Command(name, arg...)
	c.Dir = dir
	return c.CombinedOutput()
}
