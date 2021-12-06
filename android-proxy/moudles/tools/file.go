package tools

import (
	"ctp-android-proxy/configs"
	"github.com/astaxie/beego/httplib"
	"os"
	"path"
	"strings"
	"time"
)

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
	pathParams := []string{configs.CloudConfig.FileServer.LogPath, year, month, day, hour, min, sec}
	uploadPath := path.Join(pathParams...)
	req.Param("path", uploadPath)
	err = req.ToJSON(&file)
	return
}
