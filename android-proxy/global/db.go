package global

import (
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

var (
	RedisDb *redis.Client
	MysqlDb *gorm.DB
)
