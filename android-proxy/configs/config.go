package configs

import (
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
)

var CloudConfig *config

func init() {
	e := ReadConf("./configs/config.toml")
	if e != nil {
		panic("配置文件加载错误")
	}
}

type config struct {
	Server     server    `toml:"Server"`
	Redis      redisConf `toml:"Redis"`
	Mysql      mysqlConf `toml:"Mysql"`
	FileServer fileConf  `toml:"FileServer"`
}

type server struct {
	DebugMode  bool   `toml:"DebugMode"`
	HttpPort   string `toml:"HttpPort"`
	RpcPort    string `toml:"RpcPort"`
	Xcode      string `toml:"Xcode"`
	WDAPath    string `toml:"WDAPath"`
	AdbKitPath string `toml:"AdbKitPath"`
	Iproxy     string `toml:"Iproxy"`
	Host       string `toml:"Host"`
	IsIos      bool   `toml:"IsIos"`
	AdbPath    string `toml:"AdbPath"`
}

type redisConf struct {
	IpHost             string `toml:"IpHost"`
	Db                 int    `toml:"Db"`
	PoolSize           int    `toml:"PoolSize"`
	MinIdleConns       int    `toml:"MinIdleConns"`
	IdleCheckFrequency int    `toml:"IdleCheckFrequency"`
}
type fileConf struct {
	Host      string `toml:"Host"`
	Port      string `toml:"Port"`
	VideoPath string `toml:"VideoPath"`
	LogPath   string `toml:"LogPath"`
}

type mysqlConf struct {
	UserName string `toml:"UserName"`
	Password string `toml:"Password"`
	IpHost   string `toml:"IpHost"`
	DbName   string `toml:"DbName"`
}

func ReadConf(fname string) (err error) {
	var (
		fp       *os.File
		fcontent []byte
	)
	CloudConfig = new(config)
	if fp, err = os.Open(fname); err != nil {
		panic("open error " + err.Error())
		return
	}

	if fcontent, err = ioutil.ReadAll(fp); err != nil {
		panic("ReadAll error " + err.Error())
		return
	}

	if err = toml.Unmarshal(fcontent, CloudConfig); err != nil {
		panic("toml.Unmarshal error " + err.Error())
		return
	}
	return
}
