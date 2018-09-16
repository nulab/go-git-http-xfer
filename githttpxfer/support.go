package githttpxfer

import (
	"net/http"
	"os/exec"
	"strings"
	"syscall"

	"errors"
	"fmt"
)

func getServiceType(req *http.Request) string {
	serviceType := req.FormValue("service")
	if has := strings.HasPrefix(serviceType, "git-"); !has {
		return ""
	}
	return strings.Replace(serviceType, "git-", "", 1)
}

func cleanUpProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("cmd is nil")
	}
	var err error
	process := cmd.Process
	if process != nil && process.Pid > 0 {
		if e := syscall.Kill(-process.Pid, syscall.SIGTERM); e != nil {
			err = fmt.Errorf(e.Error()+" [pgid %d]", -process.Pid)
		}
	}
	cmd.Wait()
	return err
}
