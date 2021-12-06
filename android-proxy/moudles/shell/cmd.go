package shell

import (
	"bytes"
	"ctp-android-proxy/configs"
	cloudLog "ctp-android-proxy/moudles/log"
	"go.uber.org/zap"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

//新shell运行方法
type Service struct {
	Name    string
	Args    []string
	Log     *zap.Logger
	Success bool
	Pid     int
	Cmd     *exec.Cmd
	Stdout  *bytes.Buffer
	Show    bool
	Wg      *sync.WaitGroup
}

func (obj *Service) Start() error {
	var stdout bytes.Buffer
	cmd := exec.Command(obj.Name, obj.Args...)
	cmd.Stderr = &stdout
	cmd.Stdout = &stdout
	//https://www.jianshu.com/p/1f3ec2f00b03
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var err error
	obj.Stdout = &stdout
	if obj.Show {
		go obj.show()
	}
	err = cmd.Start()
	if err != nil {
		obj.Success = false
		obj.Log.Error(err.Error())
	}
	obj.Success = true
	obj.Pid = cmd.Process.Pid
	obj.Cmd = cmd
	return err
}
func (obj *Service) Run() error {
	err := obj.Start()
	if err != nil {
		return err
	}
	if obj.Wg != nil {
		obj.Wg.Done()
	}
	err = obj.Cmd.Wait()
	if err != nil {
		return err
	}
	return err
}

func (obj *Service) Stop() {
	if obj != nil {
		err := syscall.Kill(-obj.Pid, syscall.SIGKILL)
		if err != nil && err.Error() != "no such process" {
			obj.Log.Warn(err.Error())
		}
	}
}

func (obj *Service) show() {
	var oldResult string
	var newResult string
	index := 0
	for {
		if index > 60*5 {
			break
		}
		newResult = obj.Stdout.String()
		if oldResult == newResult || newResult == "" || newResult == "<nil>" {
			index += 1
			time.Sleep(time.Second)
			continue
		}
		obj.Log.Info(newResult)
		oldResult = newResult
	}
}

type Adb struct {
	Uid  string
	cmd  *Service
	Wait bool
}

func (a *Adb) Shell(args []string) (err error) {
	args = append([]string{"shell"}, args...)
	err = a.run(args)
	return err
}

func (a *Adb) Push(src string, dst string) (err error) {
	args := []string{"push", src, dst}
	err = a.run(args)
	return err
}

func (a *Adb) SetWait(status bool) {
	a.Wait = status
}

func (a *Adb) GetProp(arg string) (err error) {
	args := []string{"shell", "getprop", arg}
	err = a.run(args)
	return err
}

func (a *Adb) Result() string {
	if a.cmd.Stdout == nil {
		return ""
	}
	return a.cmd.Stdout.String()
}

func (a *Adb) Stop() {
	a.cmd.Stop()

}

func (a *Adb) run(args []string) (err error) {
	base := []string{"-s", a.Uid}
	args = append(base, args...)
	service := Service{
		Name: configs.CloudConfig.Server.AdbPath,
		Args: args,
		Log:  cloudLog.Logger,
	}
	a.cmd = &service
	if a.Wait {
		err = service.Run()
	} else {
		err = service.Start()
	}
	return
}

func (a *Adb) Forward(local string, remote string) (err error) {
	args := []string{"forward", "tcp:" + local, "tcp:" + remote}
	err = a.run(args)
	return err
}

func (a *Adb) RemoveForward(local string) (err error) {
	args := []string{"forward", "--remove", "tcp:" + local}
	err = a.run(args)
	return err
}
