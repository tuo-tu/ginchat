package router

import (
	"ginchat/docs"
	"ginchat/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// 以下为添加中文注释的代码
func Router() *gin.Engine {
	r := gin.Default()
	//swagger模块
	docs.SwaggerInfo.BasePath = "" //千万不能写成" "(也就是中间多个空格)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	//静态资源
	r.Static("/asset", "asset/")
	r.StaticFile("/favicon.ico", "asset/images/favicon.ico")
	r.LoadHTMLGlob("views/**/*")
	//首页
	r.GET("/", service.GetIndex)
	r.GET("/index", service.GetIndex)
	r.GET("/toRegister", service.ToRegister)
	r.GET("/toChat", service.ToChat)
	r.GET("/chat", service.Chat)
	r.POST("/searchFriends", service.SearchFriends)
	//用户模块
	r.POST("/user/getUserList", service.GetUserList)
	r.POST("/user/createUser", service.CreateUser)
	r.POST("/user/deleteUser", service.DeleteUser)
	r.POST("/user/updateUser", service.UpdateUser)
	r.POST("/user/findUserByNameAndPwd", service.FindUserByNameAndPwd)
	r.POST("/user/find", service.FindByID)
	//发送消息
	r.GET("/user/sendMsg", service.SendMsg)
	r.GET("/user/sendUserMsg", service.SendUserMsg)
	//添加好友
	r.POST("/contact/addfriend", service.AddFriend)
	//上传文件
	r.POST("/attach/upload", service.Upload)
	//创建群
	r.POST("/contact/createCommunity", service.CreateCommunity)
	//群列表
	r.POST("/contact/loadcommunity", service.LoadCommunity)
	r.POST("/contact/joinGroup", service.JoinGroups)
	r.POST("/user/redisMsg", service.RedisMsg)
	return r
}
