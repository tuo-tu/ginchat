# IM实时通信系统

## 项目概述

### 项目目录

ginchat总目录包括如下目录及文件

```cmd
config目录：config.yml，包含项目的配置信息；
models目录：包含一些表结构及相关的一些函数实现；
router目录：app.go，包含所有功能的路由信息；
sevice目录：主要是userservice.go，包含项目各功能的实现过程；
utils目录：实现一些必要的工具；

main.go：在main()函数里面执行初始化redis、mysql、Timer、路由引擎、启动服务；
md5.go：生成密码的算法选择；
system_init.go：包括redis和MySQL的初始化工作；
task_init.go：定时器功能的实现；
```

### 主函数

初始化相关配置（MySQL、redis等），并启动服务。

```go
func main() {
	utils.InitConfig()
	utils.InitMySQL()
	utils.InitRedis()
	InitTimer()
	r := router.Router()
	r.Run(viper.GetString("port.server"))
}

func InitTimer() {
	utils.Timer(
        time.Duration(viper.GetInt("timeout.DelayHeartbeat"))*time.Second,
		time.Duration(viper.GetInt("timeout.HeartbeatHz"))*time.Second,
		models.CleanConnection, ""
    )
}
```

### 路由分组

项目路由主要分为7组：①首页；②用户模块；③发送消息；④添加好友；⑤上传文件；⑥创建群；⑦群列表；

```go
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
```

## 用户新增

将新用户添加到数据库中，具体步骤如下：

### **路由与方法绑定**

```go
r.POST("/user/createUser", service.CreateUser)
```

### `CreateUser`函数具体实现

步骤：

1. 初始化用户基础结构体

2. 从页面中获取注册信息，并进行校验

   获取**用户名、原始密码、Identity**（和密码一样，用于密码确认，repassword）等;

3. 重复注册校验

```go
CreateUser(c *gin.Context) {
    // 1.初始化用户基础结构体
	user := models.UserBasic{}
    // 2.从页面中获取注册信息，并进行校验
	user.Name = c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	repassword := c.Request.FormValue("Identity")
	if user.Name == "" || password == "" || repassword == "" {
		c.JSON(200, gin.H{
			"code":    -1, // 0成功；-1失败
			"message": "用户名或密码不能为空",
			"data":    user,
		})
		return
	}
    // 3.重复注册校验
	if password != repassword {
		c.JSON(200, gin.H{
			"code":    -1, // 0成功；-1失败
			"message": "两次密码不一致",
			"data":    user,
		})
		return
	}
    // 4.在数据库中校验用户名是否已经被注册
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
	// 5.生成用户密码
    salt := fmt.Sprintf("%06d", rand.Int31())
	user.Salt = salt
	user.PassWord = utils.MakePassword(password, salt)
	// 6.执行新增用户
    models.CreateUser(user)
	c.JSON(200, gin.H{
		"code":    0, //0成功；-1失败
		"message": "新增用户成功",
		"data":    user,
	})
}
```

### 用户密码生成

生成需要存储在数据库中真实的密码

1. **生成salt值：**并将salt添加到user里面；

```go
salt := fmt.Sprintf("%06d", rand.Int31())
user.Salt = salt 
```

2. **生成用户密码：**`user.PassWord = utils.MakePassword(password, salt)`，MakePassword**默认使用MD5算法**对password进行加密，加密后的密码默认是小写，data的实参为(plainpwd + salt)，

3. **加密详细代码：**

```go
func Md5Encode(data string) string {
    // 1.创建一个新的 MD5 哈希对象
    h := md5.New()
    // 2.将输入数据（转换为字节）写入哈希对象
    h.Write([]byte(data))
    // 3.计算数据的最终哈希值并返回
    tempStr := h.Sum(nil)
    // 4.将结果字节切片编码为十六进制字符串并返回
    return hex.EncodeToString(tempStr)
}
```

### 加密算法改进

**MD5算法已不再被认为是加密安全的**，因此可以进行加密算法优化：

#### SHA-256算法或者SHA-512算法

将第一步改为 `h := sha256.New()`，**SHA-256**输出是 256 位，比 **MD5**（128 位）更长，从而增加了碰撞的难度。

#### **bcrypt**加密算法

通过**增加计算成本**（成本因子）来抵抗暴力破解攻击，即成本因子的值越大，算法执行的时间越长，破解密码的难度也就越大，例如：成本因子为 10 时，算法会进行 2^10 次迭代；

1. **密码哈希化**：即生成bcrypt哈希，此处使用默认的计算成本bcrypt.DefaultCost，可根据需求自己调整；

```go
func HashPassword(password string) (string, error) {
    // 生成 bcrypt 哈希
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    // 返回哈希结果
    return string(hashedPassword), nil
}
```

2. **密码验证：**

```go
func CheckPasswordHash(password, hashedPassword string) bool { 
    // 使用 bcrypt 比较密码和哈希值，关键步骤
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    return err == nil 
}
```

3. **安全性说明：**
   - **自动添加盐值（Salt）**：bcrypt 默认会在哈希中自动添加盐值（随机值），这意味着即使两个用户有相同的密码，他们的哈希值也会不同。无需手动管理盐值。
   - **bcrypt.DefaultCost**：这是计算成本的一个参数，成本越高，计算越慢，安全性越强。bcrypt.DefaultCost 的默认值为 10，适用于大多数场景。如果想要提高安全性，可以尝试增加到 12 或更高，但要注意会增加计算开销。

#### **Scrypt**加密算法

Scrypt算法比bcrypt算法更加注重内存消耗，**通过增加内存消耗来抵御暴力破解攻击**，暴力破解密码需要消耗大量的内存和计算资源。

1. **密码哈希化**：

   ```go
   func HashPasswordWithScrypt(password string) (string, error) {
       // 1.生成 32 字节的随机盐
       salt := make([]byte, 16) // 16 字节盐
       if _, err := rand.Read(salt); err != nil {
           return "", err
       }
       // 2.使用 scrypt 生成密码哈希，参数：密码、盐、N、r、p、生成的哈希长度；
   hashedPassword, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32) 
       if err != nil {
           return "", err
       }
       // 3.将盐和哈希拼接成一个字符串以便存储，以$分隔；
       return fmt.Sprintf("%x$%x", salt, hashedPassword), nil
   }
   ```

​	**N**：**内存消耗因子**，或称成本因子，决定了算法的计算复杂度。它必须是 2 的幂并且大于 1。通常建议选择 16384，作为一个平衡的值，既能提供足够的安全性，又不会占用过多的计算资源。

​	**r**：控制内存块大小和**算法的内存消耗**（块大小）。需要选择一个足够大的值来确保内存需求足以抵抗暴力破解攻击，但又不能过大，以免超出系统限制。通常保持默认值 8；

​	**p**：**并行度因子**（并行度），控制使用多少个并行进程，通常保持默认值1；

2. **密码验证：**

   ```go
   func CheckPasswordWithScrypt(password, storedHash string) bool {
       // 1.从存储的哈希中提取盐和哈希，以$分隔
       var salt, storedHashedPassword []byte
       fmt.Sscanf(storedHash, "%x$%x", &salt, &storedHashedPassword)
       // 2.使用 scrypt 重新计算哈希
       hashedPassword, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
       if err != nil {
           log.Println("Error hashing password:", err)
           return false
       }
       // 3.比较存储的哈希与计算得到的哈希
       return string(storedHashedPassword) == string(hashedPassword)
   }
   ```

3. **安全性说明：**

   - **盐值（Salt）**：与 bcrypt 一样，scrypt 会使用盐来确保即使多个用户使用相同的密码，哈希值也会不同。

   - **内存消耗**：scrypt 设计时特别注重内存的使用，这使得攻击者即使使用强大的并行计算也需要消耗大量内存，从而抵抗暴力破解。

### 执行新增用户

完成上述过程，调用`DB.Create(&user)`将新用户加入数据库，完成用户添加！

```go
models.CreateUser(user)
c.JSON(200, gin.H{
	"code":    0, //0成功；-1失败
	"message": "新增用户成功",
	"data":    user,
}
```

## 添加好友

这里仅展示**根据对方用户名**添加好友。

### 路由与方法绑定

```go
r.POST("/contact/addfriend", service.AddFriend)
```

具体代码实现：

```go
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
```

### 从页面获取信息

从上下文中获取**userid**（自己的ID）、**targetName**（对方用户名）

### 添加好友的逻辑

1. **用户名不能为空**：即targetName不能为空； 
2. **查询被添加用户**：然后根据targetName查询到对方的完整用户信息targetUser，如果targetUser.Salt为空表示未找到；
3. **排除添加自己**：不能添加自己，如果targetUser.ID == userId，响应错误 "不能添加自己"；
4. **不能重复添加**：查询自己的关系表Contact，如果contact.ID不为空，表明自己的列表中已经有这个好友，响应“不能重复添加”；

```go
func AddFriend(userId uint, targetName string) (int, string) {
	if targetName != "" {
		targetUser := FindUserByName(targetName)
		if targetUser.Salt != "" { //表示用户登陆过，在系统中存在？
			if targetUser.ID == userId {
				return -1, "不能添加自己"
			}
			contact0 := Contact{}
			utils.DB.Where("owner_id =? and target_id=? and type=1", userId, targetUser.ID).Find(&contact0)
			if contact0.ID != 0 {
				return -1, "不能重复添加"
			}
			tx := utils.DB.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			contact := Contact{}
			contact.OwnerId = userId
			contact.TargetId = targetUser.ID
			contact.Type = 1
			if err := utils.DB.Create(&contact).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			contact1 := Contact{}
			contact1.OwnerId = targetUser.ID
			contact1.TargetId = userId
			contact1.Type = 1
			if err := utils.DB.Create(&contact1).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			tx.Commit()
			return 0, "添加好友成功"
		}
		return -1, "没有找到此用户"
	}
	return -1, "好友名称不能为空"
}
```

### 执行添加好友操作

需要在双方的`Contact`（关系表）中都加入好友关系；

1. **开启事务**： gorm.DB自带的事务，如果发生panic或者事务出错，则回滚事务；

   ```go
   tx := utils.DB.Begin()
       defer func() {
          if r := recover(); r != nil || tx.Error != nil {
             tx.Rollback()
   }()
   ```

2. **在我的关系表中加入对方**：发生错误则回滚，并响应“添加好友失败”；

   ```go
   contact := Contact{}
   contact.OwnerId = userId
   contact.TargetId = targetUser.ID
   contact.Type = 1
   if err := utils.DB.Create(&contact).Error; err != nil {
   	tx.Rollback()
   	return -1, "添加好友失败"
   }
   ```

3. **在对方的关系表加入我**：发生错误则回滚，并响应“添加好友失败”；

   ```go
   contact1 := Contact{}
   contact1.OwnerId = targetUser.ID
   contact1.TargetId = userId
   contact1.Type = 1
   if err := utils.DB.Create(&contact1).Error; err != nil {
   	tx.Rollback()
   	return -1, "添加好友失败"
   }
   ```

4. **提交事务**：`tx.Commit()`，添加好友功能完成！

## 群相关功能

创建群、加载群、加入群等；

### 创建群

```go
r.POST("/contact/createCommunity", service.CreateCommunity)
```

具体代码如下：

```go
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
```

#### 从页面获取群相关信息

包括**ownerId**（群主）、**name**、**icon**（头像）、**desc**等。

#### 创建群对象

新建群结构体`Community{}`，添加上述群信息，然后执行`CreateCommunity(community)`

```go
func CreateCommunity(community Community) (int, string) {
	tx := utils.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if len(community.Name) == 0 {
		return -1, "群名称不能为空"
	}
	if community.OwnerId == 0 {
		return -1, "请先登陆"
	}
	if err := utils.DB.Create(&community).Error; err != nil {
		fmt.Println(err)
		tx.Rollback()
		return -1, "建群失败"
	}
	contact := Contact{}
	contact.OwnerId = community.OwnerId
	contact.TargetId = community.ID
	contact.Type = 2
	if err := utils.DB.Create(&contact).Error; err != nil {
		tx.Rollback()
		return -1, "添加群关系失败"
	}
	tx.Commit()
	return 0, "建群成功"
}
```

具体逻辑如下：

1. **开启事务**：捕获运行时错误，有错误则回滚：

   ```go
   tx := utils.DB.Begin()
   defer func() {
   	if r := recover(); r != nil {
   	tx.Rollback()
   }
   }()
   ```

2. **群信息校验：**

   - 群名称不能为空：`len(community.Name)`不能等于0；
   - 确保群主已经登录：`community.OwnerId`不能为空；

3. **执行创建群的操作：**有错误需要回滚，并响应“建群失败”；

   ```go
   if err := utils.DB.Create(&community).Error; err != nil {
   	fmt.Println(err)
   	tx.Rollback()
   	return -1, "建群失败"
   }
   ```

4. **创建新的关系表Contact**：注意`contact.Type = 2`（2表示群关系，1表示好友关系），,有错误则回滚，并响应“添加群关系失败”；

   ```go
   contact := Contact{}
   contact.OwnerId = community.OwnerId
   contact.TargetId = community.ID
   contact.Type = 2
   if err := utils.DB.Create(&contact).Error; err != nil {
   	tx.Rollback()
   	return -1, "添加群关系失败"
   }
   ```

5. **提交事务**：`tx.Commit()`，响应“建群成功”；

### 加载群

#### 从上下文获取群主id，即ownerId

#### 执行查询群的操作

```go
data, msg := models.LoadCommunity(uint(ownerId))
```

#### 函数具体实现如下

1. **查询关系表：**获取ownerId对应的所有关系表contacts，一个人可以拥有多个群，注意查询条件type=2；

```go
contacts := make([]Contact, 0)
utils.DB.Where("owner_id = ? and type = 2", ownerId).Find(&contacts)
```

2. **获取群id**：遍历contacts的TargetId，将群id存到objIds里面；

```go
objIds := make([]uint64, 0)
for _, v := range contacts {
	objIds = append(objIds, uint64(v.TargetId))
}
```

3. **获取群列表**：通过群id查询群列表`Communitys`，并返回，响应“查询成功”；

```go
data := make([]*Community, 10)
utils.DB.Where("id in ?", objIds).Find(&data)
for _, v := range data {
	fmt.Println(v)
}
return data, "查询成功" 
```

### 加入群

#### 从页面获取信息

从上下文中获取用户id（userId）和群id（comId）；

#### 执行加入群操作

```go
data, msg := models.JoinGroup(uint(userId), co mId)
```

函数具体实现如下：

1. **查询群：**根据群id查询群，未找到则响应“没有找到群”；

```go
community := Community{}
utils.DB.Where("id =? ", comId).Find(&community)
if community.Name == "" {
	return -1, "没有找到群"
}
```

2. **查询用户关系表**：检查用户是否有这个群；

```go
contact := Contact{}
contact.OwnerId = userId
contact.Type = 2
utils.DB.Where("owner_id =? and target_id = ? and type = 2", userId, comId).Find(&contact)
```

3. **判断用户是否需要加入群**：检查`CreatedAt`字段是否为空，若不为空，表示未加入过群，则创建新的关系表，并响应“加群成功”；

```go
if !contact.CreatedAt.IsZero() {
	return -1, "已加过此群"
} else {
	contact.TargetId = community.ID
	utils.DB.Create(&contact)
	return 0, "加群成功"
}
```

## 消息收发

即实时聊天功能，这是本项目的核心内容，**包括消息的发送和接收。**

### 路由绑定

```go
r.GET("/user/sendUserMsg", service.SendUserMsg)
```

**SendUserMsg**实际上是调用chat函数，chat函数如下：

```go
func SendUserMsg(c *gin.Context) {
	models.Chat(c.Writer, c.Request)
}
```

### 获取参数

chat函数以**c.Writer，c.Request**作为参数，函数签名为

```go
func Chat(writer http.ResponseWriter, request *http.Request)
```

### 获取用户id

即userid，与通信节点node绑定在一起，每个user都有一个通信节点node。

- **从URL中获取RawQuery参数**

```go
query := request.URL.Query()
```

RawQuery是URL中`（?）`后面的部分，是一个map结构，例如在 `http://example.com/search?q=golang&sort=desc`，RawQuery字段的值就是`“q=golang&sort=desc”`这部分。

- **从query中获取userId的值**：并将id转换为int64类型；

```go
id := query.Get("userId")
userId, _ := strconv.ParseInt(Id, 10, 64)
```

### 新建websocket连接

将**http**连接升级为**WebSocket**连接，**CheckOrigin返回true表示允许升级**，后期还可以添加防止跨站点请求伪造等功能。详细升级过程见后面；

```go
isvalida := true //表明始终通过检查
conn, err := (&websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return isvalida 
	},
}).Upgrade(writer, request, nil) // responseHeader为nil表示无需在WebSocket 握手升级的过程中添加自定义的http响应头信息。
if err != nil {
	fmt.Println(err)
	return
}
```

### 构建通信节点node

初始化一个client通信节点，每一个client绑定一个通信节点，使用map结构进行绑定，node包含以下信息；

```go
currentTime := uint64(time.Now().Unix())
node := &Node{
    Conn:          conn,
    Addr:          conn.RemoteAddr().String(), // 远端地址
    HeartbeatTime: currentTime, // 当前时间
    LoginTime:     currentTime, // 登陆时间
    DataQueue:     make(chan []byte, 50), // 节点的消息队列
GroupSets:     set.New(set.ThreadSafe), // 是线程安全的
StopChan  chan struct{} // 增加退出的信号channel
} 
```

### 将userid跟node绑定

**绑定过程需要加锁**

```go
// map结构声明在chat函数外
var clientMap map[int64]*Node = make(map[int64]*Node, 0) 
// 读写锁，声明在chat函数外
var rwlocker sync.RWMutex
// chat函数内加锁
rwlocker.Lock()
clientMap[userId] = node
rwlocker.Unlock()
```

### 发送消息

`go sendProc(node)`，**将`node.DataQueue`中的消息写入`node.Conn`**，这里增加了**消息重发机制、超时退出**（time.After）、**优雅退出**（StopChan），实现过程如下：

```go
func sendProc(node *Node) {
	for {
        select {
            // 1.将消息写入conn：从node.DataQueue中获取消息data，将数据写入conn里面；如果失败，则重发3次；
            case data := <-node.DataQueue:
            fmt.Println("sendProcMsg:", string(data))
            //将消息写入 WebSocket
            err := node.Conn.WriteMessage(websocket.TextMessage, data)
            if err != nil {
                fmt.Println("Error sending message:", err)
                retryWriteMessage(node, data) // 尝试重新发送
                return
            }
            // 2.超时退出,如果 30 秒内没有数据则超时退出；
            case <-time.After(30 * time.Second):
            fmt.Println("No data received for 30 seconds, exiting sendProc.")
            return            
            // 3.优雅推出：当接收到停止信号时，函数会优雅地退出；
            case <-node.StopChan:
            fmt.Println("Received stop signal, exiting sendProc.")
            return
        }
	}
}

// 尝试重试发送消息
func retryWriteMessage(node *Node, data []byte) {
	// 设置最大重试次数为3
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		err := node.Conn.WriteMessage(websocket.TextMessage, data)
		if err == nil {
			fmt.Println("Message sent successfully after retry.")
			return
		}
		fmt.Printf("Retry %d/%d failed: %v\n", i+1, maxRetries, err)
		time.Sleep(2 * time.Second) // 延时 2 秒再重试
	}
	// 如果重试超过最大次数依然失败，记录错误并退出
	fmt.Println("Failed to send message after retries.")
}
```

### 接收消息

`go recvProc(node)`，从`node.Conn`里面接收消息，`recvProc`实现过程如下：

#### 原始消息获取

从`node. Conn`里面接收消息；

```go
_, data, err := node.Conn.ReadMessage()
```

#### 数据解析

将消息数据解析到消息结构体`Message{}`；

```go
msg := Message{}
err = json.Unmarshal(data, &msg)
msg.CreateTime = uint64(time.Now().Unix())
```

#### 不同消息类型的处理

根据消息类型做不同的消息分发操作；

```go
switch msg.Type {
    case 1: //心跳
    currentTime := uint64(time.Now().Unix())
    node.Heartbeat(currentTime)
    fmt.Println("[ws] Heartbeat received at", currentTime)
    case 2: //私聊
    sendMsg(msg.TargetId, data)
    case 3: //群聊
    sendGroupMsg(msg.TargetId, data)
    default:
}
```

##### 心跳检测消息

主要功能是**更新node心跳**；

```go
func (node *Node) Heartbeat(currentTime uint64) {
	node.HeartbeatTime = currentTime
	return
}
```

##### 私聊消息

如果目标用户（即我，因为我接收）在线，将私聊消息放到我的node.DataQueue里面，并将私聊消息缓存到**`redis`**中，函数实现过程如下：

```go
// userId指目标用户Id
func sendMsg(userId int64, msg []byte) {
	rwlocker.RLock()
	node, ok := clientMap[userId]
	rwlocker.RUnlock()
	jsonMsg := Message{}
	json.Unmarshal(msg, &jsonMsg)
	ctx := context.Background()
	//msg里面包含了当前用户id
	jsonMsg.CreateTime = uint64(time.Now().Unix())
	userIdStr := strconv.Itoa(int(jsonMsg.UserId))
	//get key命令，
	r, err := utils.Red.Get(ctx, "online_"+userIdStr).Result()
	if err != nil {
		fmt.Println(err)
	}
	if r != "" {
		if ok {
			fmt.Println("sendMsg >>> userID: ", userId, "  msg:", string(msg))
			node.DataQueue <- msg
		}
	}
	var key string
	targetIdStr := strconv.Itoa(int(userId))
	//表示userid和targetid之间的通信消息，不区分先后，共用一个key
	if userId > jsonMsg.UserId {
		key = "msg_" + userIdStr + "_" + targetIdStr
	} else {
		key = "msg_" + targetIdStr + "_" + userIdStr
	}
	//这个函数允许cmdable类型的对象（如Redis客户端）执行ZREVRANGE命令，
	//即从Redis数据库中返回一个排序集合中指定范围内的元素，并以降序顺序返回。
	//这个命令通常用于获取范围内的分数最大的元素。
	//res, err := utils.Red.ZRevRange(ctx, key, 0, -1).Result()
	//if err != nil {
	//	fmt.Println(err)
	//}
	//score := float64(cap(res)) + 1
	score := float64(rand.Intn(100))
	//ZADD命令用于向一个有序集合（sorted set）中添加成员，并指定分数（score）
	ress, e := utils.Red.ZAdd(ctx, key, &redis.Z{score, msg}).Result()
	//res, e := utils.Red.Do(ctx, "zadd", key, 1, jsonMsg).Result() //备用 后续拓展 记录完整msg
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(ress)
}
```

1. **获取我的node**：因为我在接收消息，所以msg.TargetId就代表我，获取过程**加读锁**；

   ```go
   rwlocker.RLock()
   node, ok := clientMap[userId]
   rwlocker.RUnlock()
   ```

2. **解析数据**：将原始数据data解析到Message{}中；

   ```go
   jsonMsg := Message{}
   json.Unmarshal(msg, &jsonMsg)
   ```

3. **用户在线校验**：执行**redis**的`get key`命令，**如果从redis里面获取到了addr值，表明用户（我）在线**，则将消息放在我的消息队列中；

   ```go
   userIdStr := strconv.Itoa(int(jsonMsg.UserId))
   r, err := utils.Red.Get(ctx, "online_"+userIdStr).Result()
     // 错误处理略
   if r != "" { //表明用户在线
   if ok {
   	  fmt.Println("sendMsg >>> userID: ", userId, "  msg:", string(msg))
   	  node.DataQueue <- msg
   	}
   }
   ```

4. **消息缓存**：使用有序集合**zset结构**将收到的消息缓存到redis里面； 

   ```go
   var key string
   targetIdStr := strconv.Itoa(int(userId))
   // 1.设置表示userid和targetid之间的通信消息的key值
   if userId > jsonMsg.UserId {
   	key = "msg_" + userIdStr + "_" + targetIdStr
   } else {
   	key = "msg_" + targetIdStr + "_" + userIdStr
   }
   // 2.将消息添加到redis中
   score := float64(rand.Intn(100)) //测试用
   ress, e := utils.Red.ZAdd(ctx, key, &redis.Z{score, msg}).Result()
   ```

##### 群聊消息

这里的TargetId是指群id；

```go
func sendGroupMsg(targetId int64, msg []byte) {
	fmt.Println("开始群发消息")
	userIds := SearchUserByGroupId(uint(targetId))
	for i := 0; i < len(userIds); i++ {
		//排除自己，这里似乎有点多余
		if targetId != int64(userIds[i]) {
			sendMsg(int64(userIds[i]), msg)
		}
	}
}
```

1. **获取所有群成员id**：通过群id遍历群中的所有成员

   ```go
   userIds := SearchUserByGroupId(uint(targetId))
   ```

   函数具体实现过程如下：

   ```go
   func SearchUserByGroupId(communityId uint) []uint {
   	contacts := make([]Contact, 0)
   	objIds := make([]uint, 0)
   	// 1.获取所有群成员关系表；
   	utils.DB.Where("target_id = ? and type = 2", communityId).Find(&contacts)
   	// 2.获取群成员id：遍历contacts，获取群成员id，添加到objIds里；
   	for _, v := range contacts {
   		objIds = append(objIds, uint(v.OwnerId))
   	}
   	return objIds
   }
   ```

2. **对群成员逐个私聊**：遍历`userIds`，调用私聊函数`sendMsg`私聊所有的群成员，**但是要排除自己**；

   ```go
   for i := 0; i < len(userIds); i++ {
   	if targetId != int64(userIds[i]) {
   		sendMsg(int64(userIds[i]), msg)
   	}
   }
   ```


### 将当前用户（在线）添加到redis

**主goroutine**将当前用户添加到redis，表明当前用户是在线的，key是用户id，value是用户地址，**并设置在线时长（过期时间）**。

```go
SetUserOnlineInfo("online_"+Id，[]byte(node.Addr)，time.Duration(viper.GetInt("timeout.RedisOnlineTime"))*time.Hour)
```

具体实现：

```go
// 设置在线用户到redis
func SetUserOnlineInfo(key string, val []byte, timeTTL time.Duration) {
	ctx := context.Background()
	utils.Red.Set(ctx, key, val, timeTTL)
}
```

### 附加：Websocket升级源码解读

**总结为四个字“连接劫持”；**

```go
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Conn, error) {//代码略}
```

#### 基础信息校验

检查`WebSocket` 握手的头信息、子协议、压缩支持等； 

#### 劫持 `HTTP` 连接

```go
netConn, brw, err := h.Hijack()；
```

1. **劫持HTTP连接概念：**是指通过 `http.Hijacker`接口方法获取到**原始的网络连接对象`net.Conn`**，这个操作使得服务器可以绕**过 HTTP 协议的约束，直接操作底层的网络连接**，以便进行 WebSocket 协议所需的实时、双向通信。

2. **为什么需要劫持连接**：HTTP 连接是传统的“请求-响应”模式，而websocket需要进行持久连接，并允许双向数据流动。为了实现这一点，WebSocket 需要“劫持”HTTP 连接，将它从 HTTP 请求-响应模式中“解脱”出来，转变为一个持续的、双向的数据流连接。劫持操作返回一个底层的网络连接netConn，可以直接操作，而不再受HTTP“请求-响应”模式的限制。

#### 创建读写缓冲区

为`WebSocket`连接准备读取和写入缓存区，**读缓冲区为br，写缓冲区writeBuf；**

#### 创建WebSocket连接

创建连接并初始化其相关属性，`netConn`是劫持的网络连接，即**最原始的连接net.Conn；**

```go
c := newConn(netConn, true, u.ReadBufferSize, u.WriteBufferSize, u.WriteBufferPool, br, writeBuf)
```

#### 握手升级

握手的过程**挺抽象**； 

1. **构造HTTP 响应头**：升级请求协议、子协议、压缩扩展、responseHeader等；

2. **清除之前http连接的超时关闭操作**：

   ```go
   err := netConn.SetDeadline(time.Time{})
   ```

3. **检查是否配置了握手超时**：如果有，在规定时间内完成握手；

4. **将响应头写入连接**：

   ```go
   err = netConn.Write(p)
   ```

   将构建好的 `WebSocket` 握手响应头写入到网络连接 `netCon`中，到这一步表示握手完成！（如果设置了握手超时，握手完成后还需要清除写入超时，同第二步）

#### 返回ws连接

```go
return c, nil
```

## 文件上传

将文件上传到**本地或者阿里云**，包括图片等文件；

### 路由与方法绑定

```go
r.POST("/attach/upload", service.Upload)
```

文件上传提供两种实现，上传到本地或者阿里云；

```go
func Upload(c *gin.Context) {
	UploadLocal(c)
    // 或者UploadOSS(c)
}
```

### 上传到本地

```go
UploadLocal(c *gin.Context)
```

函数实现过程如下:

1. **读取需要上传的文件**：`srcFile`包含**文件内容**，`head`包含**文件的元数据**（比如文件名、文件大小等）;

   ```go
   w, req := c.Writer, c.Request
   //srcFile用于存储上传的文件内容，head用于存储文件名
   srcFile, head, err := req.FormFile("file")
   if err != nil {
   	utils.RespFail(w, err.Error())
   }
   ```

2. **分开文件名和后缀**：将文件的名称和后缀分开.

   ```go
   ofilName := head.Filename；
   tmp := strings.Split(ofilName, ".")
   var suffix string
   if len(tmp) > 1 {
   	suffix = "." + tmp[len(tmp)-1]
   	fmt.Println("suffix=", suffix)
   }
   ```

3. **构建新文件名**：需要指定文件后缀； 

   ```go
   fileName := fmt.Sprintf("%d%04d%s", time.Now().Unix(), rand.Int31(), suffix)
   ```

4. **创建新的空文件**：使用新文件名，**创建的文件在本地目录；**

   ```
   dstFile, err := os.Create("./asset/upload/" + fileName)
   ```

5. **复制文件**：将旧文件（即上传的文件）的内容复制给新文件；

   ```go
   _, err = io.Copy(dstFile, srcFile)
   ```

6. **响应“上传文件成功”；**

### 上传到OSS

使用**阿里云的对象存储（OSS）**进行文件上传

```go
func UploadOSS(c *gin.Context){//代码}
```

1. 读取需要上传的文件：同上；

2. 读取文件名和后缀：同上；

3. 创建新文件名：同上；

4. **创建OSS客户端**：连接OSS服务，创建OSSClient对象

   ```go
   client, err := oss.New(viper.GetString("oss.Endpoint"), viper.GetString("oss.AccessKeyId"), viper.GetString("oss.AccessKeySecret"))
   if err != nil {
   	fmt.Println("Error:", err)
   	os.Exit(-1)
   }
   ```

5. **分配存储空间**：使用client获取存储桶bucket（也称为存储空间）

   ```go
   bucket, err := client.Bucket(viper.GetString("oss.Bucket"))
   ```

6. **执行文件上传操作**：将文件`srcFile`上传到 OSS 中，并使用新文件名；

   ```go
   err = bucket.PutObject(fileName, srcFile)；
   if err != nil {
   	fmt.Println("Error:", err)
   	os.Exit(-1)
   }
   ```

7. **响应“上传文件成功”**，阿里云 OSS文件上传功能完成！

## 心跳检测下线

清理超时连接（**每个连接对应一个node用户，清理掉超时连接相当于用户下线**），使用自定义的**定时器`Timer`**（其实是调用了系统的定时器），在main函数进行初始化；

```go
func InitTimer() {
	utils.Timer(
		time.Duration(viper.GetInt("timeout.DelayHeartbeat"))*time.Second,
		time.Duration(viper.GetInt("timeout.HeartbeatHz"))*time.Second,
		models.CleanConnection, "")
}
```

Timer()函数实现过程如下：

```go
type TimerFunc func(interface{}) bool
// delay：首次延迟，delay时间后开始执行定时器；
// tick：每隔 tick 时间调用一次函数 fun
// fun：定时执行的方法，返回bool值，用于控制是否要结束此定时器；
// param：方法fun的参数
func Timer(delay, tick time.Duration, fun TimerFunc, param interface{}) {
	go func() {
		if fun == nil {
			return
		}
		t := time.NewTimer(delay) //定时器
		for {
			select {
			case <-t.C:
				if fun(param) == false {
					return
				}
				t.Reset(tick)
			}
		}
	}()
}
```

1. **开启goroutine**：将定时器放在一个goroutine里面；

2. **检测是否需要启动timer**：如果 `fun == nil`，即没有定时执行的函数，不启动定时器； 

3. **初始化定时器**

   ```go
   t := time.NewTimer(delay)
   ```

   表示这个定时器在**`delay`**后开始执行；

4. **检查定时器是否到期**：使用 for 循环**不断检查定时器是否到期**，当定时器到期时，触发 `t.C` 的信号，执行回调函数 `fun(param)`，

   - 如果回调函数返回 false，**停止定时器**，不再继续执行。

   - 如果回调函数返回 true，则直接**重置定时器**，时间间隔为tick，表明定时器在tick时间后再次触发（间隔周期本来就是tick）。

5. **定时器定时执行的函数fun为清理超时连接**：

   ```go
   CleanConnection(param interface{}) (result bool)
   ```

   **（1）使用defer 语句来捕获 panic**，如果发生异常（比如 **`node.Conn.Close()`** 出现问题），会打印错误信息，并继续执行：

   ```go
   defer func() {
   	if r := recover(); r != nil {
   		fmt.Println("清理超时连接异常", r)
   	}
   }()
   ```

   **（2）遍历每一个客户端通信节点node**，如果心跳检测超时：

   ```go
   node.HeartbeatTime+viper.GetUint64("timeout.HeartbeatMaxTime") <= currentTime）
   ```

   即node节点的心跳时间加上最大超时时间小于当前时间，则关闭websocket连接；

   ```go
   for i := range clientMap {
   	node := clientMap[i]
   	if node.IsHeartbeatTimeOut(currentTime) {
   		fmt.Println("心跳超时，关闭连接", node)
   		node.Conn.Close()
   	}
   }
   ```

## NewTimer源码解读：

```go
func NewTimer(d Duration) *Timer {
	c := make(chan Time, 1)
	t := &Timer{
		C: c,
		r: runtimeTimer{
			when: when(d),
			f:    sendTime,
			arg:  c,
		},
	}
	startTimer(&t.r)
	return t
}
```

### 创建time通道

用于接收定时器超时的事件；

```go
c := make(chan Time, 1)；
```

### 初始化Timer

构造了一个新的 Timer 对象

```go
t := &Timer{
    C: c, 					// 用于接收超时事件。
	r: runtimeTimer{			// 负责处理定时器的运行逻辑。
        when: when(d),		// 定时器触发的时间
		f:    sendTime, 		// 唤醒时被调用的函数, sendTim负责将当前时间发送到通道c
		arg:  c, 				// sendTim函数的参数, 
    },
} 
```

### 启动定时器

`startTimer(&t.r)`启动定时器，但最终是通过调用`runtime/time.go`文件中的**addtimer函数**实现此过程，**`addtimer`函数将`Timer`添加当前`P`的定时器队列中**，参数`t`是`runtime. timer`类型。函数的具体实现如下：

```go
func addtimer(t *timer) {
	// 基础校验，代码略
	t.status.Store(timerWaiting)
	when := t.when
	mp := acquirem()
	pp := getg().m.p.ptr()
	lock(&pp.timersLock)
	cleantimers(pp)
	doaddtimer(pp, t)
	unlock(&pp.timersLock)
	wakeNetPoller(when)
	releasem(mp)
}
```

1. **定时器参数校验**：①定时器触发时间必须为正值；②定时器的周期必须为非负值；③定时器要无状态（timerNoStatus），不能已经被初始化；

2. **初始化定时器状态**：`t.status.Store(timerWaiting)`，timerWaiting表示该定时器正在等待被调度（也表示等待被添加到P的定时器队列中）；

3. **获取M并禁用抢占**：`mp := acquirem()`，获取当前的M（即Machine，M也表示系统线程），并禁用抢占，确保在操作P中的定时器队列时，当前的goroutine不会被抢占，避免并发环境下出现竞态条件；
   - **goroutine 被抢占**指的是**正在执行任务的 goroutine 被调度器暂停（抢占）**，并且控制权转移到其他 goroutine 上。

4. **获取P**：`pp := getg().m.p.ptr()`，获取当前正在执行的goroutine所绑定的处理器；

5. **锁定P的定时器队列：**`lock(&pp.timersLock)`，pp.timers实际上是一个最小堆，确保在操作P上的timer时，其他goroutine无法同时访问或修改该队列；

6. **清理P定时器队列的头部**：`cleantimers(pp)`，因为头部定时器最先到期，即清理`pp.timers[0]`，确保定时器的状态得到正确的更新，并按需要进行移除或重新安排。

7. **将当前timer添加到P的定时器队列中**：`doaddtimer(pp, t)`，最小堆的根节点总是最早到期的定时器，它会优先触发。

   - `t.pp.set(pp)`：是将定时器分配给处理器P，

   - `pp.timers = append(pp.timers, t)`：将定时器添加到P的定时器队列的尾部，

   - `siftupTimer(pp.timers, i)`：调整定时器的位置，维护 timer 在P 的最小堆中的位置。

8. **解锁P的定时器队列**：`unlock(&pp.timersLock)`，解锁后允许其他 goroutine 访问此定时器队列。

9. **唤醒空闲的P**：`wakeNetPoller(when)`；唤醒在网络轮询器中休眠的线程，其实就是唤醒一个空闲的P，使其能够处理新的定时器事件； 

10. **释放M并恢复抢占**：`releasem(mp)`；这意味着定时器队列操作完成后，系统可以再次允许goroutine被抢占。

### 返回定时器

```go
return t
```

 
