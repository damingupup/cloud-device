/*
* @Author: 于智明
* @Date:   2021/2/3 3:52 下午
 */
package configs

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"os"
)

var CloudConfig *config

func Init() {
	namespaceId := os.Getenv("NamespaceId")
	dataId := os.Getenv("DataId")
	group := os.Getenv("Group")
	fmt.Println(namespaceId)
	fmt.Println(dataId)
	fmt.Println(group)
	clientConfig := constant.ClientConfig{
		NamespaceId:         namespaceId, // 如果需要支持多namespace，我们可以场景多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "logs/log",
		CacheDir:            "logs/cache",
		RotateTime:          "1h",
		MaxAge:              3,
		LogLevel:            "debug",
	}
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			"nacos.qa.com",
			80,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		panic(err.Error())
		return
	}
	content, err := configClient.GetConfig(vo.ConfigParam{DataId: dataId, Group: group})
	if len(content) == 0 {
		panic("配置中心读取失败")
	}
	fmt.Println(content)
	err = ReadConf(content)
	if err != nil {
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
	DebugMode bool   `toml:"DebugMode"`
	HttpPort  string `toml:"HttpPort"`
	RpcPort   string `toml:"RpcPort"`
	Xcode     string `toml:"Xcode"`
	WDAPath   string `toml:"WDAPath"`
	Iproxy    string `toml:"Iproxy"`
	Host      string `toml:"Host"`
	IsIos     bool   `toml:"IsIos"`
	Python    string `toml:"Python"`
	AdbPath   string `toml:"AdbPath"`
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
}

type mysqlConf struct {
	UserName string `toml:"UserName"`
	Password string `toml:"Password"`
	IpHost   string `toml:"IpHost"`
	DbName   string `toml:"DbName"`
}

func ReadConf(content string) (err error) {
	CloudConfig = new(config)
	if err = toml.Unmarshal([]byte(content), CloudConfig); err != nil {
		panic("toml.Unmarshal error " + err.Error())
		return
	}
	return
}
