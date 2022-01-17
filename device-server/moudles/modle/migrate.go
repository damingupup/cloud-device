/*
* @Author: 于智明
* @Date:   2021/2/23 3:51 下午
 */
package model

import cloudLog "ctp-device-server/moudles/log"

func Migrate() {
	err := mysqlDb().AutoMigrate(&FileModel{}, &DeviceModel{})
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
}
