package model

import cloudLog "ctp-android-proxy/moudles/log"

func Migrate() {
	err := mysqlDb().AutoMigrate(&FileModel{}, &DeviceModel{})
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
}
