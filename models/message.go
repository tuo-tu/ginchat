package models

import (
	"context"
	"encoding/json"
	"fmt"
	"ginchat/utils"
	"github.com/fatih/set"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// 消息
type Message struct {
	gorm.Model
	UserId     int64  //发送者
	TargetId   int64  //接收者
	Type       int    //发送类型 1群聊 2私聊 3心跳
	Media      int    //消息类型 1文字 2表情包 3图片 4音频
	Content    string //消息内容
	CreateTime uint64 //创建时间
	ReadTime   uint64 //读取时间
	Pic        string
	Url        string
	Desc       string
	Amount     int //其他数字统计
}

func (table *Message) TableName() string {
	return "message"
}

type Node struct {
	Conn          *websocket.Conn //连接
	Addr          string          //客户端地址
	FirstTime     uint64          //首次连接时间
	HeartbeatTime uint64          //心跳时间
	LoginTime     uint64          //登陆时间
	DataQueue     chan []byte
	GroupSets     set.Interface
	StopChan      chan struct{}
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwlocker sync.RWMutex

// 需要：发送者ID、接收者ID，消息类型，发送者的内容，发送类型
func Chat(writer http.ResponseWriter, request *http.Request) {
	//1.校验token
	query := request.URL.Query()
	Id := query.Get("userId")
	userId, _ := strconv.ParseInt(Id, 10, 64)
	isvalida := true //checkToken() 待......
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return isvalida //来源检查
		},
	}).Upgrade(writer, request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	//2.获取token
	currentTime := uint64(time.Now().Unix())
	node := &Node{
		Conn:          conn,
		Addr:          conn.RemoteAddr().String(), //本人的客户端地址？应该是远端地址吧
		HeartbeatTime: currentTime,                //心跳时间
		LoginTime:     currentTime,                //登陆时间
		DataQueue:     make(chan []byte, 50),
		GroupSets:     set.New(set.ThreadSafe), //这是线程安全的
	}
	//3.用户关系
	//4.userid跟node绑定并加锁
	rwlocker.Lock()
	clientMap[userId] = node
	rwlocker.Unlock()
	//sync.Map{}
	//5、完成发送的逻辑
	go sendProc(node)
	//6、完成接收逻辑
	go recvProc(node)
	//7、加入在线用户到缓存，应该做一个盘点，已经关闭连接的用户不能加入redis
	SetUserOnlineInfo("online_"+Id, []byte(node.Addr), time.Duration(viper.GetInt("timeout.RedisOnlineTime"))*time.Hour)
}

// 发送过程
/*func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			fmt.Println("[ws]sendProc >>>> msg :", string(data))
			//将消息写进conn里面
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}*/

func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			// 打印发送的消息
			fmt.Println("sendProcMsg:", string(data))
			// 尝试写入消息到 WebSocket
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				// 连接错误处理
				fmt.Println("Error sending message:", err)
				// 尝试重新连接或重试的逻辑
				// 例如调用重连函数或等待
				retryWriteMessage(node, data)
				return
			}
		case <-time.After(30 * time.Second):
			// 如果 30 秒内没有数据则超时退出
			fmt.Println("No data received for 30 seconds, exiting sendProc.")
			return
		case <-node.StopChan:
			// 接收到停止信号，优雅退出
			fmt.Println("Received stop signal, exiting sendProc.")
			return
		}
	}
}

// 尝试重试发送消息
func retryWriteMessage(node *Node, data []byte) {
	// 可设置最大重试次数
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

// 接收过程调度
func recvProc(node *Node) { // node是我？
	for {
		_, data, err := node.Conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		msg := Message{}
		err = json.Unmarshal(data, &msg)
		if err != nil {
			fmt.Println(err)
		}
		msg.CreateTime = uint64(time.Now().Unix())
		switch msg.Type {
		case 3: //心跳
			currentTime := uint64(time.Now().Unix())
			node.Heartbeat(currentTime)
			fmt.Println("[ws] Heartbeat received at", currentTime)
		case 1: //私聊
			sendMsg(msg.TargetId, data)
		case 2: //群聊
			sendGroupMsg(msg.TargetId, data)
		default:
			// 异步处理消息，以免阻塞接收
			//go func(data []byte) {
			//Dispatch(data)
			//	BroadMsg(data)
			//	fmt.Println("[ws] recvProc <<<<< ", string(data))
			//}(data)
		}

	}
}

var udpsendChan chan []byte = make(chan []byte, 1024)

func BroadMsg(data []byte) {
	udpsendChan <- data
}

func init() {
	go udpSendProc()
	go udpRecvProc()
	fmt.Println("init goroutine ")
}

// 完成udp数据发送goroutine
func udpSendProc() {
	con, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(192, 168, 43, 1),
		Port: viper.GetInt("port.udp"),
	})
	defer con.Close()
	if err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case data := <-udpsendChan:
			fmt.Println("udpSendProc  data :", string(data))
			//将消息写进conn里面
			_, err1 := con.Write(data)
			if err1 != nil {
				fmt.Println(err1)
				return
			}
		}
	}
}

// 完成udp数据接收goroutine
func udpRecvProc() {
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: viper.GetInt("port.udp"),
	})
	if err != nil {
		fmt.Println(err)
	}
	defer con.Close()
	for {
		var buf [512]byte //为什么要这样定义
		n, err1 := con.Read(buf[0:])
		if err1 != nil {
			fmt.Println(err1)
			return
		}
		fmt.Println("udpRecvProc  data :", string(buf[0:n]))
		Dispatch(buf[0:n])
	}
}

// 后端调度逻辑处理
func Dispatch(data []byte) {
	msg := Message{}
	msg.CreateTime = uint64(time.Now().Unix())
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Println(err)
		return
	}
	switch msg.Type {
	case 1: //私信，id是指好友id
		fmt.Println("dispatch  data :", string(data))
		sendMsg(msg.TargetId, data)
	case 2: //群发，这里id是指群id
		sendGroupMsg(msg.TargetId, data)
		//	sendGroupMsg()
		//case 3: //广播
		//	sendAllMsg()
	}
}

func JoinGroup(userId uint, comId string) (int, string) {
	community := Community{}
	utils.DB.Where("id =? or name =?", comId, comId).Find(&community)
	if community.Name == "" {
		return -1, "没有找到群"
	}
	contact := Contact{}
	contact.OwnerId = userId
	contact.Type = 2
	utils.DB.Where("owner_id =? and target_id = ? and type = 2", userId, comId).Find(&contact)
	if !contact.CreatedAt.IsZero() {
		return -1, "已加过此群"
	} else {
		contact.TargetId = community.ID
		utils.DB.Create(&contact)
		return 0, "加群成功"
	}
}

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

// 需要重写此方法才能完整的msg转byte[]
func (msg Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(msg)
}

// 获取缓存里面的消息
func RedisMsg(userIdA, userIdB, start, end int64, isRev bool) []string {
	ctx := context.Background()
	userIdStr := strconv.Itoa(int(userIdA))
	targetIdStr := strconv.Itoa(int(userIdB))
	var key string
	if userIdA > userIdB {
		key = "msg_" + targetIdStr + "_" + userIdStr
	} else {
		key = "msg_" + userIdStr + "_" + targetIdStr
	}
	var rels []string
	var err error
	if isRev {
		rels, err = utils.Red.ZRange(ctx, key, start, end).Result()
	} else {
		rels, err = utils.Red.ZRevRange(ctx, key, start, end).Result()
	}
	if err != nil {
		fmt.Println(err)
	}
	return rels
}

// targetId是指群id或者好友id
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

// 更新用户心跳
func (node *Node) Heartbeat(currentTime uint64) {
	node.HeartbeatTime = currentTime
	return
}

// 清理超时连接
func CleanConnection(param interface{}) (result bool) {
	result = true //意义何在？
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("清理超时连接异常", r)
		}
	}()
	currentTime := uint64(time.Now().Unix())
	//遍历每一个node
	for i := range clientMap {
		node := clientMap[i]
		if node.IsHeartbeatTimeOut(currentTime) {
			fmt.Println("心跳超时...关闭连接", node)
			node.Conn.Close()
		}
	}
	return result
}

// 用户心跳是否超时
func (node *Node) IsHeartbeatTimeOut(currentTime uint64) (timeout bool) {
	if node.HeartbeatTime+viper.GetUint64("timeout.HeartbeatMaxTime") <= currentTime {
		fmt.Println("心跳超时。。。自动下线", node)
		timeout = true
	}
	return
}
