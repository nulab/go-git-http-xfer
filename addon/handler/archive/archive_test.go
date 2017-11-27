package archive

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/nulab/go-git-http-xfer/githttpxfer"
)

func Test_it_should_download_archive_repository(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		t.Log("git is not found. so skip git archive test.")
		return
	}

	ghx, err := githttpxfer.New("/data/git", "/usr/bin/git")
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
	os.Mkdir(absRepoPath, os.ModeDir)

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

	if _, err := execCmd(destDir, "git", "push", "-u", "origin", "master"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "wget", "-O-", remoteRepoUrl+"/archive/master.zip"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

	if _, err := execCmd(destDir, "wget", "-O-", remoteRepoUrl+"/archive/master.tar"); err != nil {
		t.Errorf("execute command error: %s", err.Error())
		return
	}

}

func execCmd(dir string, name string, arg ...string) ([]byte, error) {
	c := exec.Command(name, arg...)
	c.Dir = dir
	return c.CombinedOutput()
}
