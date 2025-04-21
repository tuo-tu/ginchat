package service

import (
	"fmt"
	"ginchat/models"
	"ginchat/utils"
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// GetUserList
// @Summary 所有用户
// @Tags 用户模块
// @Success 200 {string} json{"code","message"}
// @Router /user/getUserList [get]
func GetUserList(c *gin.Context) {
	data := make([]*models.UserBasic, 10)
	data = models.GetUserList()
	c.JSON(200, gin.H{
		"message": data,
	})
	c.JSON(200, gin.H{
		"code": 0, //0成功；-1失败
		//"message": "用户名已注册",
		"data": data,
	})
}

// CreateUser
// @Summary 新增用户
// @Tags 用户模块
// @param name query string false "用户名"
// @param password query string false "密码"
// @param repassword query string false "确认密码"
// @Success 200 {string} json{"code","message"}
// @Router /user/createUser [get]
func CreateUser(c *gin.Context) {
	//user.Name = c.Query("name")
	//password := c.Query("password")
	//repassword := c.Query("repassword")
	user := models.UserBasic{}
	user.Name = c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	repassword := c.Request.FormValue("Identity")
	if user.Name == "" || password == "" || repassword == "" {
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "用户名或密码不能为空",
			"data":    user,
		})
		return
	}
	if password != repassword {
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "两次密码不一致",
			"data":    user,
		})
		return
	}
	fmt.Println(user.Name, "  >>>>>>>>>>>  ", password, repassword)
	data := models.FindUserByName(user.Name)
	if data.Name != "" {
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "用户名已注册",
			"data":    user,
		})
		return
	}
	salt := fmt.Sprintf("%06d", rand.Int31())
	user.Salt = salt
	//user.PassWord = password
	user.PassWord = utils.MakePassword(password, salt)
	models.CreateUser(user)
	c.JSON(200, gin.H{
		"code":    0, //0成功；-1失败
		"message": "新增用户成功",
		"data":    user,
	})
}

// FindUserByNameAndPwd
// @Summary 所有用户
// @Tags 用户模块
// @param name query string false "用户名"
// @param password query string false "密码"
// @Success 200 {string} json{"code","message"}
// @Router /user/findUserByNameAndPwd [post]
func FindUserByNameAndPwd(c *gin.Context) {
	data := models.UserBasic{}
	//name := c.Query("name")
	//password := c.Query("password")
	name := c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	user := models.FindUserByName(name)
	if user.Name == "" {
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "该用户不存在",
			"data":    data,
		})
		return
	}
	if flag := utils.ValidPassword(password, user.Salt, user.PassWord); !flag {
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "密码不正确",
			"data":    data,
		})
		return
	}
	pwd := utils.MakePassword(password, user.Salt)
	data = models.FindUserByNameAndPwd(name, pwd)
	c.JSON(200, gin.H{
		"code":    0, //0成功；-1失败
		"message": "登陆成功",
		"data":    data,
	})
}

// DeleteUser
// @Summary 删除用户
// @Tags 用户模块
// @param id query string false "id"
// @Success 200 {string} json{"code","message"}
// @Router /user/deleteUser [get]
func DeleteUser(c *gin.Context) {
	user := models.UserBasic{}
	id, _ := strconv.Atoi(c.Query("id"))
	user.ID = uint(id)
	models.DeleteUser(user)
	c.JSON(200, gin.H{
		"code":    0, //0成功；-1失败
		"message": "删除用户成功",
		"data":    user,
	})
}

//注意：formData不要写成formDate

// UpdateUser
// @Summary 修改用户
// @Tags 用户模块
// @param id formData string false "id"
// @param name formData string false "name"
// @param password formData string false "password"
// @param phone formData string false "phone"
// @param email formData string false "email"
// @Success 200 {string} json{"code","message"}
// @Router /user/updateUser [post]
func UpdateUser(c *gin.Context) {
	user := models.UserBasic{}
	id, _ := strconv.Atoi(c.PostForm("id"))
	user.ID = uint(id)
	user.Name = c.PostForm("name")
	user.PassWord = c.PostForm("password")
	user.Phone = c.PostForm("phone")
	user.Avatar = c.PostForm("icon")
	user.Email = c.PostForm("email")
	fmt.Println("update :", user)
	if _, err := govalidator.ValidateStruct(user); err != nil {
		fmt.Println(err)
		c.JSON(200, gin.H{
			"code":    -1, //0成功；-1失败
			"message": "修改参数不匹配",
			"data":    user,
		})
	} else {
		//校验通过再更新
		models.UpdateUser(user)
		c.JSON(200, gin.H{
			"code":    0, //0成功；-1失败
			"message": "修改用户成功",
			"data":    user,
		})
	}
}

// 防止跨域站点伪造请求,
// Upgrader指定将HTTP连接升级为WebSocket连接的参数
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 测试用？
func SendMsg(c *gin.Context) {
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(ws *websocket.Conn) {
		err = ws.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(ws)
	MsgHandeler(ws, c)
}

func RedisMsg(c *gin.Context) {
	//处理HTTP POST请求中的表单数据
	userIdA, _ := strconv.Atoi(c.PostForm("userIdA"))
	userIdB, _ := strconv.Atoi(c.PostForm("userIdB"))
	start, _ := strconv.Atoi(c.PostForm("start"))
	end, _ := strconv.Atoi(c.PostForm("end"))
	//默认return false
	isRev, _ := strconv.ParseBool(c.PostForm("isRev"))
	res := models.RedisMsg(int64(userIdA), int64(userIdB), int64(start), int64(end), isRev)
	utils.RespOKList(c.Writer, "ok", res)
}

func MsgHandeler(ws *websocket.Conn, c *gin.Context) {
	for {
		msg, err := utils.Subscrible(c, utils.PublishKey)
		if err != nil {
			fmt.Println(err)
		}
		tm := time.Now().Format("2006-01-02 15:04:05")
		m := fmt.Sprintf("[ws][%s]:%s", tm, msg)
		//这个函数目前只处理写（write）的情况
		err = ws.WriteMessage(1, []byte(m))
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// SendUserMsg 函数用于处理用户消息
func SendUserMsg(c *gin.Context) {
	// 调用models.Chat函数处理用户消息
	models.Chat(c.Writer, c.Request)
}

func SearchFriends(c *gin.Context) {
	id, _ := strconv.Atoi(c.Request.FormValue("userId"))
	users := models.SearchFriend(uint(id))
	utils.RespOKList(c.Writer, users, len(users))
}

func AddFriend(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Request.FormValue("userId"))
	targetName := c.Request.FormValue("targetName")
	code, msg := models.AddFriend(uint(userId), targetName)
	if code == 0 {
		utils.RespOK(c.Writer, code, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

func CreateCommunity(c *gin.Context) {
	ownerId, _ := strconv.Atoi(c.Request.FormValue("ownerId"))
	name := c.Request.FormValue("name")
	icon := c.Request.FormValue("icon")
	desc := c.Request.FormValue("desc")
	community := models.Community{}
	community.OwnerId = uint(ownerId)
	community.Name = name
	community.Img = icon
	community.Desc = desc
	code, msg := models.CreateCommunity(community)
	if code == 0 {
		utils.RespOK(c.Writer, code, msg)
	} else {
		//c.Writer用于写入HTTP响应的缓冲区
		utils.RespFail(c.Writer, msg)
	}
}

func LoadCommunity(c *gin.Context) {
	ownerId, _ := strconv.Atoi(c.Request.FormValue("ownerId"))
	data, msg := models.LoadCommunity(uint(ownerId))
	if len(data) != 0 {
		utils.RespList(c.Writer, 0, data, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

func JoinGroups(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Request.FormValue("userId"))
	comId := c.Request.FormValue("comId")
	data, msg := models.JoinGroup(uint(userId), comId)
	if data == 0 {
		utils.RespOK(c.Writer, data, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

func FindByID(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Request.FormValue("userId"))
	data := models.FindByID(uint(userId))
	utils.RespOK(c.Writer, data, "ok")
}
