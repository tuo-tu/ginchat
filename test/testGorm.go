package main

import (
	"ginchat/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 迁移 schema
	//这样写无法创建的原因是？这里只要第一次执行就可以，建好了以后直接用create添加记录
	//这样写无法连接db, err := gorm.Open(mysql.Open(viper.GetString("mysql.dns")), &gorm.Config{})
	tmpStr := "root:123456@tcp(127.0.0.1:3306)/ginchat?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(tmpStr), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败")
	}
	db.AutoMigrate(&models.UserBasic{}, &models.Message{}, &models.GroupBasic{}, &models.Contact{}, &models.Community{})
	// Create
	//user := &models.UserBasic{}
	//牢记，最好不要将创建和查询同时使用，会报错如下等情况；
	/*[1.481ms] [rows:0] SELECT * FROM `user_basic` WHERE `user_basic`.`id` = 3 AND `u
	  ser_basic`.`deleted_at` IS NULL AND `user_basic`.`id` = 4 ORDER BY `user_basic`.
	  	`id` LIMIT 1
	  	2024/04/04 23:41:16 record not found
	*/
	//db.Create(user)
	//表示查询第一条或最后一条主键为2的记录，这说明也许有多个2的情况，但是只返回一个
	/*if err := utils.DB.First(user, 4).Error; err != nil {
		log.Println(err)
	}*/

	//注意，这里是返回查询到的信息的值，即前面一定先要有个查询，
	//查询到的值用user接收，相当于复用user了
	//fmt.Println(user.ID)
	//db.First(user, "code = ?", "D42") // 查找 code 字段值为 D42 的记录

	// Update - 将 product 的 price 更新为 200
	//utils.DB.Model(user).Update("PassWord", "123456")
	// Update - 更新多个字段
	//db.Model(user).Updates() // 仅更新非零值字段
	//db.Model(user).Updates(map[string]interface{}{"Price": 200, "Code": "F42"})

	// Delete - 删除 product
	//db.Delete(user, 1)
}
