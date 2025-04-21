package service

import (
	"ginchat/models"
	"github.com/gin-gonic/gin"
	"strconv"
	"text/template"
)

// GetIndex
// @Tags 首页
// @Success 200 {string} welcome
// @Router /index [get]
func GetIndex(c *gin.Context) { //前面的注释和这里之间不能有空格
	ind, err := template.ParseFiles("index.html", "views/chat/head.html")
	if err != nil {
		panic(err)
	}
	ind.Execute(c.Writer, "index")
}

func ToRegister(c *gin.Context) {
	ind, err := template.ParseFiles("views/user/register.html")
	if err != nil {
		panic(err)
	}
	ind.Execute(c.Writer, "register")
}

// ToChat函数接收一个*gin.Context 类型的参数 c，它代表当前的HTTP请求上下文。
func ToChat(c *gin.Context) {
	ind, err := template.ParseFiles(
		"views/chat/index.html",
		"views/chat/head.html",
		"views/chat/foot.html",
		"views/chat/tabmenu.html",
		"views/chat/concat.html",
		"views/chat/group.html",
		"views/chat/profile.html",
		"views/chat/createcom.html",
		"views/chat/userinfo.html",
		"views/chat/main.html")
	if err != nil {
		panic(err)
	}
	userId, _ := strconv.Atoi(c.Query("userId"))
	token := c.Query("token")
	user := models.UserBasic{}
	user.ID = uint(userId)
	user.Identity = token
	ind.Execute(c.Writer, user)
}

func Chat(c *gin.Context) {
	models.Chat(c.Writer, c.Request)
}
