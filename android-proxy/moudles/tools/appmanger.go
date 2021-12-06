package tools

import (
	cloudLog "ctp-android-proxy/moudles/log"
	"ctp-android-proxy/moudles/shell"
	"fmt"
	"github.com/pkg/errors"
	"strings"
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
	adb := shell.Adb{Uid: am.Uid, Wait: true}
	snowId := SnowId()
	name := "/data/local/tmp/" + snowId + "tmp.apk"
	defer func() {
		adb.Shell([]string{"rm", "-rf", name})
	}()
	adb.Push(am.Path, name)
	fmt.Println(adb.Result())
	adb.Shell([]string{"pm", "install", name})
	result := strings.TrimSpace(adb.Result())
	if !strings.Contains(result, "Success") {
		cloudLog.Logger.Error(result)
		return errors.New(result)
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
