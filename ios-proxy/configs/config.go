package configs

import (
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
)

func init() {
	e := ReadConf("./configs/config.toml")
	if e != nil {
		panic("配置文件加载错误")
	}
}

var ConfigiOS *config

type config struct {
	Server     server    `toml:"Server"`
	Mysql      mysqlConf `toml:"Mysql"`
	FileServer fileConf  `toml:"FileServer"`
	Env        env       `toml:"Env"`
}
type server struct {
	DebugMode bool   `toml:"DebugMode"`
	HttpPort  string `toml:"HttpPort"`
	Xcode     string `toml:"Xcode"`
	WDAPath   string `toml:"WDAPath"`
	Iproxy    string `toml:"Iproxy"`
	Host      string `toml:"Host"`
	Python    string `toml:"Python"`
}
type mysqlConf struct {
	UserName string `toml:"UserName"`
	Password string `toml:"Password"`
	IpHost   string `toml:"IpHost"`
	DbName   string `toml:"DbName"`
}
type fileConf struct {
	Host    string `toml:"Host"`
	Port    string `toml:"Port"`
	LogPath string `toml:"LogPath"`
}

type env struct {
	IPASigner string `toml:"IPASigner"`
}

func ReadConf(name string) (err error) {
	var (
		fp      *os.File
		content []byte
	)
	ConfigiOS = new(config)
	if fp, err = os.Open(name); err != nil {
		panic("open error " + err.Error())
		return
	}

	if content, err = ioutil.ReadAll(fp); err != nil {
		panic("ReadAll error " + err.Error())
		return
	}

	if err = toml.Unmarshal(content, ConfigiOS); err != nil {
		panic("toml.Unmarshal error " + err.Error())
		return
	}
	return
}
