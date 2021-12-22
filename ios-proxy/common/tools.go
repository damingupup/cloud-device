package common

import (
	"bytes"
	"context"
	"github.com/astaxie/beego/httplib"
	"go.uber.org/zap"
	"ctp-ios-proxy/configs"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

func GetAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		Log.Error(err.Error())
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

type CmdUtil struct {
	Path    string //指令
	Args    []string
	Dir     string //工作目录
	Pid     int
	Success bool
	Log     *zap.Logger
	Running bool
	Cmd     exec.Cmd
	Out     []byte
}

func (obj *CmdUtil) Exec() {
	var out bytes.Buffer
	var stderr bytes.Buffer
	obj.Cmd = exec.Cmd{
		Path:   obj.Path,
		Dir:    obj.Dir,
		Args:   obj.Args,
		Stdout: &out,
		Stderr: &stderr,
	}
	obj.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := obj.Cmd.Start()
	if err != nil {
		obj.Log.Error(obj.Path)
		obj.Log.Error("指令执行失败", zap.Strings("msg", []string{err.Error(), stderr.String()}))
		obj.Success = false
	}
	obj.Pid = obj.Cmd.Process.Pid
}

func (obj *CmdUtil) Stop() {
	err := syscall.Kill(-obj.Pid, syscall.SIGKILL)
	if err != nil && err.Error() != "no such process" {
		Log.Warn(err.Error())
	}
	obj.Pid = 0
}

func GetFreePort() (port int) {
	listener, _ := net.Listen("tcp4", "0.0.0.0:0")
	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return
}

func PathExists(path string) bool {

	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

//新shell运行方法
type CmdService struct {
	Name    string
	Args    []string
	Log     *zap.Logger
	Success bool
	Pid     int
	Cmd     *exec.Cmd
	Context context.Context
}

func (obj *CmdService) Run() {
	cmd := exec.Command(obj.Name, obj.Args...)
	//https://www.jianshu.com/p/1f3ec2f00b03
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Start()
	if err != nil {
		obj.Success = false
		obj.Log.Error(err.Error())
	}
	obj.Success = true
	obj.Pid = cmd.Process.Pid
	obj.Cmd = cmd

}

func (obj *CmdService) Stop() {
	err := syscall.Kill(-obj.Pid, syscall.SIGKILL)
	if err != nil {
		obj.Log.Warn(err.Error())
	}
}

type FileResult struct {
	Url     string `json:"url"`
	Md5     string `json:"md5"`
	Path    string `json:"path"`
	Domain  string `json:"domain"`
	Scene   string `json:"scene"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
	Scenes  string `json:"scenes"`
	Retmsg  string `json:"retmsg"`
	Retcode int    `json:"retcode"`
	Src     string `json:"src"`
}

func UploadFile(filepath string, file *FileResult) (err error) {
	uploadUrl := strings.Join([]string{"http://", configs.ConfigiOS.FileServer.Host, ":",
		configs.ConfigiOS.FileServer.Port, "/group1/upload"}, "")
	req := httplib.Post(uploadUrl)
	req.PostFile("file", filepath) //注意不是全路径
	req.Param("output", "json")
	req.Param("scene", "")
	date := time.Now()
	year := date.Format("2006")
	month := date.Format("01")
	day := date.Format("02")
	hour := date.Format("15")
	min := date.Format("04")
	sec := date.Format("5")
	pathParams := []string{configs.ConfigiOS.FileServer.LogPath, year, month, day, hour, min, sec}
	uploadPath := path.Join(pathParams...)
	req.Param("path", uploadPath)
	err = req.ToJSON(&file)
	return
}
