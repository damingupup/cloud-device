package model

import (
	"ctp-android-proxy/configs"
	"ctp-android-proxy/global"
	"ctp-android-proxy/moudles/log"
	"fmt"
	"github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

func Init() {
	cloudlog.Logger.Info("数据库初始化")
	global.MysqlDb = mysqlDb()
	redisConfig := configs.CloudConfig.Redis
	global.RedisDb = redis.NewClient(&redis.Options{
		Addr:               redisConfig.IpHost, // use default Addr
		Password:           "",                 // no password set
		DB:                 redisConfig.Db,     // use default DB
		PoolSize:           redisConfig.PoolSize,
		MinIdleConns:       redisConfig.MinIdleConns,
		IdleCheckFrequency: time.Duration(redisConfig.IdleCheckFrequency) * time.Second,
	})
	//Migrate()
}

func mysqlDb() *gorm.DB {
	mysqlConfig := configs.CloudConfig.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true&loc=Local",
		mysqlConfig.UserName,
		mysqlConfig.Password,
		mysqlConfig.IpHost,
		mysqlConfig.DbName,
	)
	sqlDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		cloudlog.Logger.Error("数据库连接炸了啊")
		panic(err)
	}
	dbPool, err := sqlDB.DB()
	if err != nil {
		cloudlog.Logger.Error("数据库连接池炸了啊")
		panic(err)
	}
	dbPool.SetMaxIdleConns(10)
	dbPool.SetMaxOpenConns(100)
	dbPool.SetConnMaxLifetime(2 * time.Hour)
	cloudlog.Logger.Info("数据库连接成功")
	return sqlDB

}
