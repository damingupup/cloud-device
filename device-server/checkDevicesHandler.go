/*
* @Author: 于智明
* @Date:   2021/2/3 5:18 下午
 */
package main

import (
	"ctp-device-server/configs"
	"ctp-device-server/global"
	cloudLog "ctp-device-server/moudles/log"
	model "ctp-device-server/moudles/modle"
	"ctp-device-server/moudles/shell"
	"encoding/json"
	"fmt"
	mapSet "github.com/deckarep/golang-set"
	"github.com/go-redis/redis"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Device struct {
	OnlineDevices mapSet.Set
	DB            *gorm.DB
	RedisClient   *redis.Client
}

func (d *Device) diffData(nowDevices mapSet.Set) (addData mapSet.Set, removeData mapSet.Set) {
	removeData = d.OnlineDevices.Difference(nowDevices)
	addData = nowDevices.Difference(d.OnlineDevices)
	return
}

const (
	deviceOnline  = 0 //空闲
	deviceOffline = 2
	deviceDamage  = 3
	deviceDebug   = 4
	deviceBorrow  = 5
)

func (d *Device) initDevice(uid string) {
	var device model.DeviceModel
	d.DB.Where(&model.DeviceModel{SerialId: uid}).First(&device)
	Id := device.ID
	//没有设备则添加
	if Id == 0 {
		if configs.CloudConfig.Server.IsIos {
			device = model.DeviceModel{SerialId: uid, StatusId: deviceDebug, System: global.Ios, IsCloud: true}
		} else {
			//安卓记录部分详细信息
			device = d.createAndroidDevice(uid)
		}
		tx := d.DB.Create(&device)
		if tx.RowsAffected != 1 {
			cloudLog.Logger.Error("添加设备失败" + uid)
			return
		}
		cloudLog.Logger.Info("设备入库成功" + uid)
	} else {
		if !configs.CloudConfig.Server.IsIos {
			androidVersion := d.getAndroidVersion(uid)
			if device.SystemMsg != androidVersion {
				device.SystemMsg = androidVersion
				d.DB.Save(device)
			}
		}
	}
	excludeStatus := [2]int{deviceDebug, deviceBorrow}
	d.DB.Model(&device).Where("statuId not in ? and isDelete = false", excludeStatus).Update("statuId", deviceOnline)
	//present为0则设备离线
	msg := map[string]interface{}{
		"present":    global.OnLine,
		"current_ip": configs.CloudConfig.Server.Host,
		"rpc_port":   configs.CloudConfig.Server.RpcPort,
	}
	res, err := d.RedisClient.HMSet(uid, msg).Result()
	if err != nil {
		cloudLog.Logger.Error("设备redis连接失败" + uid)
		return
	}
	if !strings.Contains(res, "OK") {
		cloudLog.Logger.Error("设备redis连接失败" + uid)
		return
	}
	d.RedisClient.Set("init_timestamp", time.Now().UnixNano()/1e6, 0)
	cloudLog.Logger.Info("redis记录成功" + uid)
	return
}

func (d *Device) removeDevice(uid string) {
	d.RedisClient.Del(uid)
	msg := map[string]interface{}{"present": global.OffLine}
	d.RedisClient.HMSet(uid, msg)
	d.RedisClient.Set("init_timestamp", time.Now().UnixNano()/1e6, 0)
	d.DB.Model(&model.DeviceModel{}).Where("serialId = ? and statusId = ?", uid, deviceOnline).Update("statuId", deviceOffline)
}

func (d *Device) CheckDevice() {
	for {
		var cmd string
		var params []string
		if configs.CloudConfig.Server.IsIos {
			cmd = "idevice_id"
			params = []string{}
		} else {
			cmd = configs.CloudConfig.Server.AdbPath
			params = append(params, "devices")
		}
		result := shell.Service{Name: cmd, Args: params}
		result.Run()
		deviceDataStr := result.Stdout.String()
		var deviceData []string
		if configs.CloudConfig.Server.IsIos {
			deviceData = strings.Split(deviceDataStr, " (USB)\n")
		} else {
			deviceData = d.handleAndroidDevice(deviceDataStr)
		}
		time.Sleep(300 * time.Millisecond)
		deviceUid := mapSet.NewSet()
		for _, uid := range deviceData {
			if uid == "" {
				continue
			}
			deviceUid.Add(uid)
		}
		addDevice, removeDevice := d.diffData(deviceUid)
		for uid := range addDevice.Iter() {
			d.OnlineDevices.Add(uid)
			cloudLog.Logger.Info("新增设备" + uid.(string))
			go d.initDevice(uid.(string))
		}
		for uid := range removeDevice.Iter() {
			d.OnlineDevices.Remove(uid)
			cloudLog.Logger.Info("移除设备" + uid.(string))
			d.removeDevice(uid.(string))
		}
	}
}

func (d *Device) handleAndroidDevice(data string) []string {
	devices := []string{}
	tmp := strings.Split(data, "\n")
	for _, i := range tmp {
		if strings.Contains(i, "List of devices attached") || !strings.Contains(i, "device") {
			continue
		}
		devices = append(devices, strings.Replace(i, "\tdevice", "", 1))
	}
	return devices
}

func (d *Device) createAndroidDevice(uid string) (device model.DeviceModel) {
	adb := shell.Adb{Uid: uid, Wait: true}
	androidVersion := d.getAndroidVersion(uid)
	adb.Shell([]string{"pm", "list", "packages", "-3"})
	result := adb.Result()
	result = strings.Replace(result, "package:", "", -1)
	result = strings.TrimSpace(result)
	apps := strings.Split(result, "\n")
	size := d.getAndroidSize(uid)
	dataBytes, err := json.Marshal(apps)
	var data string
	if err != nil {
		data = "[]"
	} else {
		data = string(dataBytes)
	}
	return model.DeviceModel{SerialId: uid, StatusId: deviceDebug, SystemMsg: androidVersion, System: global.Android, Apk: data, Resolution: size}
}

func (d *Device) updateAndroidVersion(uid string) {
	androidVersion := d.getAndroidVersion(uid)
	size := d.getAndroidSize(uid)
	fmt.Println(androidVersion)
	fmt.Println(size)
}

func (d *Device) getAndroidVersion(uid string) string {
	adb := shell.Adb{Uid: uid, Wait: true}
	adb.Shell([]string{"getprop", "ro.build.version.release"})
	androidVersion := adb.Result()
	androidVersion = "Android " + strings.TrimSpace(androidVersion)
	return androidVersion
}

func (d *Device) getAndroidSize(uid string) string {
	adb := shell.Adb{Uid: uid, Wait: true}
	adb.Shell([]string{"wm", "size"})
	sizeInfo := adb.Result()
	var size string
	for _, v := range strings.Split(sizeInfo, "\n") {
		if strings.HasPrefix(v, "Physical") {
			size = strings.Replace(v, "Physical size: ", "", 1)
			size = strings.TrimSpace(size)
			index := strings.Index(size, "x")
			if index < 0 {
				continue
			}
			size = size[(index+1):] + "x" + size[:index]
		}
	}
	return size
}
