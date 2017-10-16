package githttptransfer

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
)

func Test_Git_getRequestFileInfo_should_return_RequestFileInfo(t *testing.T) {

	gitRootPath, err := ioutil.TempDir("", "githttptransfer")
	if err != nil {
		t.Errorf("Create Temp Dir error: %s", err.Error())
		return
	}
	defer os.RemoveAll(gitRootPath)

	git := newGit(gitRootPath, "/usr/bin/git", true, true)

	repoPath := "foo"
	filePath := "README.txt"

	absRepoPath := git.GetAbsolutePath(repoPath)
	os.Mkdir(absRepoPath, os.ModeDir)

	touchCmd := exec.Command("touch", filePath)
	touchCmd.Dir = absRepoPath

	if _, err := touchCmd.CombinedOutput(); err != nil {
		t.Errorf("Touch Coummand error: %s", err.Error())
	}

	if _, err := git.GetRequestFileInfo(repoPath, filePath); err != nil {
		t.Errorf("RequestFileInfo is not exists. error: %s", err.Error())
	}

}

func Test_Git_getRequestFileInfo_should_not_return_RequestFileInfo(t *testing.T) {

	gitRootPath, err := ioutil.TempDir("", "githttptransfer")
	if err != nil {
		t.Errorf("Create Temp Dir error: %s", err.Error())
		return
	}
	defer os.RemoveAll(gitRootPath)

	git := newGit(gitRootPath, "/usr/bin/git", true, true)

	repoPath := "foo"
	filePath := "README.txt"

	if _, err := git.GetRequestFileInfo(repoPath, filePath); err == nil {
		t.Errorf("RequestFileInfo is exists. error: %s", err.Error())
	}

}

func Test_Git_exists_should_return_true_if_exists_repository(t *testing.T) {

	gitRootPath, err := ioutil.TempDir("", "githttptransfer")
	if err != nil {
		t.Errorf("Create Temp Dir error: %s", err.Error())
		return
	}
	defer os.RemoveAll(gitRootPath)

	git := newGit(gitRootPath, "/usr/bin/git", true, true)

	repoPath := "foo"

	os.Mkdir(path.Join(gitRootPath, repoPath), os.ModeDir)

	if !git.Exists(repoPath) {
		t.Errorf("this repository is not exists. path: %s", git.GetAbsolutePath(repoPath))
	}

}

func Test_Git_exists_should_return_false_if_not_exists_repository(t *testing.T) {

	gitRootPath, err := ioutil.TempDir("", "githttptransfer")
	if err != nil {
		t.Errorf("Create Temp Dir error: %s", err.Error())
		return
	}
	defer os.RemoveAll(gitRootPath)

	git := newGit(gitRootPath, "/usr/bin/git", true, true)

	repoPath := "foo"

	if git.Exists(repoPath) {
		t.Errorf("this repository is exists. path: %s", git.GetAbsolutePath(repoPath))
	}

}

func Test_Git_getAbsolutePath_should_return_absolute_path_of_git_repository(t *testing.T) {

	gitRootPath, err := ioutil.TempDir("", "githttptransfer")
	if err != nil {
		t.Errorf("Create Temp Dir error: %s", err.Error())
		return
	}
	defer os.RemoveAll(gitRootPath)

	git := newGit(gitRootPath, "/usr/bin/git", true, true)

	repoPath := "foo"
	expectedPath := path.Join(gitRootPath, repoPath)
	resultPath := git.GetAbsolutePath(repoPath)
	if expectedPath != resultPath {
		t.Errorf("path is not %s . result: %s", expectedPath, resultPath)
	}

}
