package main

import (
	"demo/common/utils/consul"
	"demo/user/global"
	"demo/user/initialize"
	"flag"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"os"
	"os/signal"
	"syscall"

	"demo/user/conf"
	"demo/user/server/grpc"
	"demo/user/server/http"
	log "github.com/sirupsen/logrus"
)

var HttpAddr string
var GrpcAddr string

func init() {
	// flags
	// go run user/main.go -grpc-host="0.0.0.0" -grpc-port=5001 -http-host="0.0.0.0" -http-port=8081
	flag.StringVar(&conf.HttpHost, "http-host", "0.0.0.0", "http服务IP/域名")
	flag.IntVar(&conf.HttpPort, "http-port", 8080, "http服务端口")
	flag.StringVar(&conf.GrpcHost, "grpc-host", "0.0.0.0", "grpc服务IP/域名")
	flag.IntVar(&conf.GrpcPort, "grpc-port", 5000, "grpc服务端口")

	flag.Parse()

	HttpAddr = fmt.Sprintf("%s:%d", conf.HttpHost, conf.HttpPort)
	GrpcAddr = fmt.Sprintf("%s:%d", conf.GrpcHost, conf.GrpcPort)

	// log
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2005-01-02 15:04:05",
	})

	log.WithFields(log.Fields{
		"http-addr": HttpAddr,
		"grpc-addr": GrpcAddr,
	}).Info("run flags:")

	// 初始化
	initialize.InitConfig()
	initialize.InitLogger()
}

func main() {
	errc := make(chan error)

	// http server
	{
		log.WithField("http-addr", HttpAddr).Info("http server is running...")
		go http.Run(HttpAddr, errc)
	}

	// grpc server
	{
		log.WithField("grpc-addr", GrpcAddr).Info("grpc server is running...")
		go grpc.Run(GrpcAddr, errc)
	}

	// consul 服务注册
	client := consul.NewClient("consul", 8500)
	rc := consul.NewRegistryClient(client)
	serviceId := fmt.Sprintf("%s", uuid.NewV4())
	tags := []string{"srv"}
	//err := rc.Register("golang", conf.HttpPort, global.ServerConfig.Name, tags, serviceId, "http")
	//err := rc.Register("golang", conf.GrpcPort, global.ServerConfig.Name, tags, serviceId, "GRPC")
	err := rc.RegisterByGrpc("golang", conf.GrpcPort, global.ServerConfig.Name, tags, serviceId)
	if err != nil {
		panic(err)
	}

	// 监听终止信号
	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-quit)
	}()

	log.WithField("error", <-errc).Info("Exit")

	// 服务反注册
	if err := rc.DeRegister(serviceId); err != nil{
		log.Info("服务从consul中：注销失败")
	} else {
		log.Info("服务从consul中：注销成功")
	}
}
