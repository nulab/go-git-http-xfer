package githttptransfer

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
)

func newGit(rootPath string, binPath string, uploadPack bool, receivePack bool) *git {
	return &git{rootPath, binPath, uploadPack, receivePack}
}

type git struct {
	rootPath    string
	binPath     string
	uploadPack  bool
	receivePack bool
}

func (g *git) HasAccess(req *http.Request, rpc string, checkContentType bool) bool {
	if checkContentType {
		if req.Header.Get("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
			return false
		}
	}
	if rpc == receivePack {
		return g.receivePack
	}
	if rpc == uploadPack {
		return g.uploadPack
	}
	return false
}

func (g *git) GetAbsolutePath(repoPath string) string {
	return path.Join(g.rootPath, repoPath)
}

func (g *git) Exists(repoPath string) bool {
	absRepoPath := g.GetAbsolutePath(repoPath)
	if _, err := os.Stat(absRepoPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (g *git) GetInfoRefs(repoPath, serviceName string) ([]byte, error) {
	args := []string{serviceName, "--stateless-rpc", "--advertise-refs", "."}
	return g.GitCommand(repoPath, args...).Output()
}

func (g *git) UpdateServerInfo(repoPath string) ([]byte, error) {
	args := []string{"update-server-info"}
	return g.GitCommand(repoPath, args...).Output()
}

func (g *git) GitCommand(repoPath string, args ...string) *exec.Cmd {
	command := exec.Command(g.binPath, args...)
	command.Dir = g.GetAbsolutePath(repoPath)
	return command
}

func (g *git) GetRequestFileInfo(repoPath, filePath string) (*RequestFileInfo, error) {
	absRepoPath := g.GetAbsolutePath(repoPath)
	absFilePath := path.Join(absRepoPath, filePath)
	info, err := os.Stat(absFilePath)
	if err != nil {
		return nil, err
	}
	return &RequestFileInfo{info, absFilePath}, nil
}

type RequestFileInfo struct {
	os.FileInfo
	AbsolutePath string
}
