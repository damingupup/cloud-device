/*
* @Author: 于智明
* @Date:   2021/2/22 3:17 下午
 */
package model

import "ios-proxy/common"

func Migrate() {
	db := common.GetMysql()
	err := db.AutoMigrate(&FileModel{})
	if err != nil {
		common.Log.Error(err.Error())
	}
}
