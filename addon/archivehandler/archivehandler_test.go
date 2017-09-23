package archivehandler

import (
	"testing"
	"os/exec"
	"net/http/httptest"
	"io/ioutil"
	"path"
	"os"

	"github.com/vvatanabe/go-git-http-transfer/githttptransfer"
)

func Test_it_should_download_archive_repository(t *testing.T) {

	_, err := exec.LookPath("git")
	if err != nil {
		t.Log("git is not found. so skip git archive test.")
		return
	}

	ght := githttptransfer.New("/data/git", "/usr/bin/git", true, true)
	ght.AddRoute(githttptransfer.NewRoute(
		Method,
		Pattern,
		New(ght).HandlerFunc,
	))

	ts := httptest.NewServer(ght)
	if ts == nil {
		t.Error("test server is nil.")
	}
	defer ts.Close()

	repoName := "archive_test.git"
	absRepoPath := ght.Git.GetAbsolutePath(repoName)
	os.Mkdir(absRepoPath, os.ModeDir)

	initCmd := exec.Command("git", "init", "--bare", "--shared")
	initCmd.Dir = absRepoPath
	testCommand(t, initCmd)

	remoteUrl := ts.URL + "/" + repoName

	tempDir, _ := ioutil.TempDir("", "gitsmarthttp")
	dir := "archive_test"
	destDir := path.Join(tempDir, dir)

	cloneCmd := exec.Command("git", "clone", remoteUrl, destDir)
	testCommand(t, cloneCmd)

	setUserNameToGitConfigCmd := exec.Command("git", "config", "--global", "user.name", "yuichi.watanabe")
	setUserNameToGitConfigCmd.Dir = destDir
	testCommand(t, setUserNameToGitConfigCmd)

	setUserEmailToGitConfigCmd := exec.Command("git", "config", "--global", "user.email", "yuichi.watanabe.ja@gmail.com")
	setUserEmailToGitConfigCmd.Dir = destDir
	testCommand(t, setUserEmailToGitConfigCmd)

	touchCmd := exec.Command("touch", "README.txt")
	touchCmd.Dir = destDir
	testCommand(t, touchCmd)

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = destDir
	testCommand(t, addCmd)

	commitCmd := exec.Command("git", "commit", "-m", "first commit")
	commitCmd.Dir = destDir
	testCommand(t, commitCmd)

	pushCmd := exec.Command("git", "push", "-u", "origin", "master")
	pushCmd.Dir = destDir
	testCommand(t, pushCmd)


	wgetZipCmd := exec.Command("wget", "-O-", remoteUrl + "/archive/master.zip")
	initCmd.Dir = destDir
	testCommand(t, wgetZipCmd)

	wgetTarCmd := exec.Command("wget", "-O-", remoteUrl + "/archive/master.tar")
	initCmd.Dir = destDir
	testCommand(t, wgetTarCmd)

}

func testCommand(t *testing.T, cmd *exec.Cmd) {
	out, err := cmd.CombinedOutput()
	t.Logf("%s: %s", cmd.Args, string(out))
	if err != nil {
		t.Errorf("testCommand error: %s", err.Error())
	}
}
