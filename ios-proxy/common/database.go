package common

import (
	"fmt"
	"ctp-ios-proxy/configs"

	//"github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var sqlDB *gorm.DB
func GetMysql() *gorm.DB {
	return sqlDB
}
func InItMysql()  {
	sqlDB = MysqlDb()
}

func MysqlDb() *gorm.DB {
	mysqlConfig := configs.ConfigiOS.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local",
		mysqlConfig.UserName,
		mysqlConfig.Password,
		mysqlConfig.IpHost,
		mysqlConfig.DbName,
	)
	var err error
	sqlDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		Log.Error("数据库连接炸了啊")
		panic(err)
	}
	dbPool, err := sqlDB.DB()
	if err != nil {
		Log.Error("数据库连接池炸了啊")
		panic(err)
	}
	dbPool.SetMaxIdleConns(10)
	dbPool.SetMaxOpenConns(100)
	dbPool.SetConnMaxLifetime(2 * time.Hour)
	Log.Info("数据库连接成功")
	return sqlDB

}

