/*
* @Author: 于智明
* @Date:   2021/2/23 3:40 下午
 */

package model

import (
	"time"
)

type Model struct {
	ID         int32     `gorm:"column:id;not null;autoIncrement;primaryKey;"`
	CreateTime time.Time `gorm:"column:createtime;type:timestamp;comment:创建时间;default:CURRENT_TIMESTAMP"`
	UpdateTime time.Time `gorm:"column:updatetime;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP(0);comment:修改时间"`
	IsDelete   bool      `gorm:"column:isDelete;type:smallint; default:0"`
	Msg        string    `gorm:"column:msg;type:varchar(255);default:'';comment:备注"`
}
type FileModel struct {
	Model
	UserId   int    `gorm:"column:userId;type:int(11);comment:用户id"`
	FiledId  int    `gorm:"column:filedId;type:int(11);comment:存储id"`
	GroupId  int    `gorm:"column:groupId;type:int(11);comment:组别id"`
	Domain   string `gorm:"column:domain;type:varchar(255);default:'';comment:文件domain"`
	Md5Id    string `gorm:"column:md5Id;type:varchar(255);default:'';comment:文件md5"`
	Name     string `gorm:"column:name;type:varchar(255);default:'';comment:文件名称"`
	Path     string `gorm:"column:path;type:varchar(255);default:'';comment:存储路径"`
	Type     string `gorm:"column:type;type:varchar(255);default:'';comment:文件类型，log，视频，图片，app"`
	LogId    int    `gorm:"column:logId;type:int(11);comment:使用记录id"`
	Size     string `gorm:"column:size;type:varchar(255);default:'';comment:文件大小"`
	DeviceId int    `gorm:"column:deviceId;type:int(11);default:0;comment:上传设备"`
}

func (FileModel) TableName() string {
	return "tb_file"
}
