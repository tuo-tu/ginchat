package models

import (
	"fmt"
	"ginchat/utils"
	"gorm.io/gorm"
	"time"
)

type UserBasic struct {
	gorm.Model
	Name          string
	PassWord      string
	Phone         string `valid:"matches(^1[3-9]{1}\\d{9}$)"`
	Email         string `valid:"email"`
	Avatar        string //头像
	Identity      string
	ClientIp      string
	ClientPort    string
	Salt          string
	LoginTime     *time.Time `gorm:"log_out_time" json:"login_out_time"`
	HeartbeatTime *time.Time
	LoginOutTime  *time.Time
	IsLogout      bool
	DeviceInfo    string
}

func (table *UserBasic) TableName() string {
	return "user_basic"
}

func GetUserList() []*UserBasic {
	data := make([]*UserBasic, 10)
	utils.DB.Find(&data)
	return data
}

func FindUserByNameAndPwd(name, password string) UserBasic {
	user := UserBasic{}
	//将查到的记录存放在user里面
	utils.DB.Where("name = ? and pass_word = ?", name, password).First(&user)
	//token加密
	str := fmt.Sprintf("%d", time.Now().Unix())
	tmp := utils.Md5Encode(str)
	//更新操作,每次调用都更新identity
	utils.DB.Model(&user).Where("id = ?", user.ID).Update("identity", tmp)
	return user
}

func FindUserByName(name string) UserBasic {
	user := UserBasic{}
	utils.DB.Where("name = ?", name).First(&user)
	return user
}

func FindUserByPhone(phone string) *gorm.DB {
	user := UserBasic{}
	return utils.DB.Where("phone = ?", phone).First(&user)
}

func FindUserByEmail(email string) *gorm.DB {
	user := UserBasic{}
	return utils.DB.Where("email = ?", email).First(&user)
}

func CreateUser(user UserBasic) *gorm.DB {
	return utils.DB.Create(&user)
}

func DeleteUser(user UserBasic) *gorm.DB {
	return utils.DB.Delete(&user)
}

func UpdateUser(user UserBasic) *gorm.DB {
	return utils.DB.Model(&user).Updates(
		UserBasic{
			Name:     user.Name,
			PassWord: user.PassWord,
			Phone:    user.Phone,
			Email:    user.Email,
			Avatar:   user.Avatar,
		})
}

// 查找某个用户
func FindByID(id uint) UserBasic {
	user := UserBasic{}
	utils.DB.Where("id = ?", id).First(&user)
	return user
}
