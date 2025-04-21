package models

import (
	"gorm.io/gorm"
)

// 组
type GroupBasic struct {
	gorm.Model
	Name    string
	OwnerId uint
	Icon    string
	Type    int //对应的类型，0，1，2
}

func (table *GroupBasic) TableName() string {
	return "group_basic"
}
