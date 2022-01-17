/*
* @Author: 于智明
* @Date:   2021/2/3 5:35 下午
 */
package global

import (
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

var (
	RedisDb *redis.Client
	MysqlDb *gorm.DB
)
