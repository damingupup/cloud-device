/*
* @Author: 于智明
* @Date:   2021/2/24 7:48 下午
 */
package model

type DeviceModel struct {
	Model
	StatusId     int    `gorm:"column:statusId;type:int(11);default:0;comment:状态id0空闲，1占用，2，离线，3损坏，4调试，5，占用,调试状态需要隐藏设备"`
	SerialId     string `gorm:"column:serialId;uniqueIndex;type:varchar(255);default:'';comment:手机序列号"`
	Name         string `gorm:"column:name;type:varchar(50);default:'';comment:手机名称"`
	Brand        string `gorm:"column:brand;type:varchar(30);default:'';comment:品牌"`
	Cpu          string `gorm:"column:cpu;type:varchar(50);default:'';comment:cpu"`
	Memory       string `gorm:"column:memory;type:varchar(20);default:'';comment:内存"`
	Size         string `gorm:"column:size;type:varchar(20);default:'';comment:手机屏幕尺寸"`
	Resolution   string `gorm:"column:resolution;type:varchar(20);default:'';comment:屏幕分辨率"`
	MacAddress   string `gorm:"column:macAddress;type:varchar(30);default:'';comment:mac地址"`
	Root         bool   `gorm:"column:root;type:smallint; default:0;comment:是否root"`
	System       int    `gorm:"column:system;type:int(11);comment:操作系统，1为安卓2为ios"`
	SystemMsg    string `gorm:"column:systemMsg;type:varchar(255);default:'';comment:操作系统详细信息"`
	PicId        string `gorm:"column:picId;type:varchar(255);default:'';comment:手机图片"`
	FrameId      int    `gorm:"column:frameId;type:int(11);comment:手机边框"`
	Network      string `gorm:"column:network;type:varchar(255);default:'';comment:网络类型"`
	Apk          string `gorm:"column:apk;type:text;comment:手机初始化数据"`
	BorrowStatus int    `gorm:"column:borrow_status;type:int(11);comment:借用状态"`
	IsCloud      bool   `gorm:"column:is_cloud;type:smallint; default:0;comment:是否云测机器"`
}

func (DeviceModel) TableName() string {
	return "tb_device"
}
