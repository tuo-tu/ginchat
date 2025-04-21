package utils

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

var (
	Red *redis.Client
	DB  *gorm.DB
)

func InitConfig() {
	//viper.SetConfigFile("config/app.yml")
	viper.SetConfigName("app")
	viper.AddConfigPath("config")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
	}
	fmt.Println("config app inited")
}

func InitRedis() {
	Red = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.DB"),
		PoolSize:     viper.GetInt("redis.poolSize"),
		MinIdleConns: viper.GetInt("redis.minIdleConn"),
	})
}

func InitMySQL() {
	//自定义日志模板，打印SQL语句
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second, //慢SQL的阀值
			Colorful:      true,        //彩色
			LogLevel:      logger.Info, //级别
		})

	DB, _ = gorm.Open(mysql.Open(viper.GetString("mysql.dns")), &gorm.Config{Logger: newLogger})
	fmt.Println("MySQL inited")
}

const PublishKey = "websocket" //常量声明

// 发布消息到redis
func Publish(ctx context.Context, channel string, msg string) error {
	//var err error
	fmt.Println("Publish...", msg)
	err := Red.Publish(ctx, channel, msg).Err()
	if err != nil {
		fmt.Println(err)
	}
	return err //nil表示没有错误
}

// 订阅
func Subscrible(ctx context.Context, channel string) (string, error) {
	sub := Red.Subscribe(ctx, channel)
	fmt.Println("Subscrible...", ctx)
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	fmt.Println("Subscrible...", msg.Payload)
	return msg.Payload, err //Payload是什么？
}
