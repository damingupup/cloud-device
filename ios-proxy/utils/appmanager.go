/*
* @Author: 于智明
* @Date:   2021/1/11 5:27 下午
 */
package utils

import (
	"bytes"
	"ios-proxy/common"
	"os/exec"
)

import (
	"github.com/pkg/errors"
	"regexp"
)

var canFixedInstallFails = map[string]bool{
	"INSTALL_FAILED_PERMISSION_MODEL_DOWNGRADE": true,
	"INSTALL_FAILED_UPDATE_INCOMPATIBLE":        true,
	"INSTALL_FAILED_VERSION_DOWNGRADE":          true,
}

type APKManager struct {
	Path         string
	Uid          string
	packageName  string
	mainActivity string
}

func (am *APKManager) Install() error {
	params := []string{"-u", am.Uid, "-i", am.Path}
	cmd := exec.Command("/usr/local/bin/ideviceinstaller", params...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Run()
	common.Log.Info(b.String())
	matches := regexp.MustCompile(`ERROR:.+`).FindStringSubmatch(b.String())
	if len(matches) > 0 {
		common.Log.Error(matches[0])
		return errors.New(matches[0])
	}
	matches = regexp.MustCompile(`WARNING:.+`).FindStringSubmatch(b.String())
	if len(matches) > 0 {
		common.Log.Warn(matches[0])
		//return errors.New(matches[0])
	}
	if err != nil {
		common.Log.Error(err.Error())
		return err
	}
	return nil
}

//ForceInstall install apk
func (am *APKManager) ForceInstall() error {
	return am.Install()
}

type StartOptions struct {
	Stop bool
	Wait bool
}

//func (am *APKManager) Start(opts StartOptions) error {
//
//	if am.mainActivity == "" {
//		return errors.New("parse MainActivity failed")
//	}
//	mainActivity := am.mainActivity
//	if !strings.Contains(mainActivity, ".") {
//		mainActivity = "." + mainActivity
//	}
//	_, err = runShellTimeout(30*time.Second, "am", "start", "-n", packageName+"/"+mainActivity)
//	return err
//}
