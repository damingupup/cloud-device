/*
* @Author: 于智明
* @Date:   2021/2/3 5:19 下午
 */
package tools

import (
	"ctp-device-server/configs"
	"github.com/astaxie/beego/httplib"
	"net"
	"path"
	"strings"
	"time"
)

func GetFreePort() (port int) {
	listener, _ := net.Listen("tcp4", "0.0.0.0:0")
	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return
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

func UploadFile(filepath string, file *FileResult) error {
	uploadUrl := strings.Join([]string{"http://", configs.CloudConfig.FileServer.Host, ":",
		configs.CloudConfig.FileServer.Port, "/group1/upload"}, "")
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
	pathParams := []string{configs.CloudConfig.FileServer.VideoPath, year, month, day, hour, min, sec}
	uploadPath := path.Join(pathParams...)
	req.Param("path", uploadPath)
	err := req.ToJSON(&file)
	return err
}
