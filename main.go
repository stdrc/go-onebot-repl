package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strings"

	libob "github.com/botuniverse/go-libonebot"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type OneBotDummy struct {
	*libob.OneBot
	config        *DummyConfig
	lastMessageID uint64
}

type Config struct {
	OneBot libob.Config `mapstructure:",squash"`
	Dummy  DummyConfig
}

type DummyConfig struct {
	SelfID string `mapstructure:"self_id"`
	UserID string `mapstructure:"user_id"`
}

//go:embed default_config.toml
var defaultConfigString string

func loadConfig() *Config {
	v := viper.New()
	v.SetConfigType("toml")
	v.ReadConfig(strings.NewReader(defaultConfigString))
	v.SetConfigFile("config.toml")
	err := v.MergeInConfig()
	if err != nil && os.IsNotExist(err) {
		fmt.Println("配置文件不存在, 正在写入默认配置到 config.toml")
		v.WriteConfigAs("config.toml")
	}
	config := &Config{}
	v.Unmarshal(config)
	fmt.Printf("配置加载成功: %+v\n", config)
	return config
}

func main() {
	config := loadConfig()
	ob := &OneBotDummy{
		OneBot:        libob.NewOneBot("dummy", &config.OneBot),
		config:        &config.Dummy,
		lastMessageID: 0,
	}
	logFile, err := os.OpenFile("dummy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	ob.Logger.SetOutput(logFile)
	ob.Logger.SetLevel(logrus.InfoLevel)

	mux := libob.NewActionMux()
	mux.HandleFunc(libob.ActionGetVersion, ob.handleGetVersion)
	mux.HandleFunc(libob.ActionGetSelfInfo, ob.handleGetSelfInfo)
	mux.HandleFunc(libob.ActionSendMessage, ob.handleSendMessage)
	mux.HandleFuncExtended("test", func(w libob.ResponseWriter, r *libob.Request) {
		w.WriteData("It works!")
	})

	ob.Handle(mux)
	go ob.Run()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("请开始对话 (输入 exit 退出):")
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "exit" {
			ob.Shutdown()
			break
		}
		go ob.Push(&libob.MessageEvent{
			Event: libob.Event{
				SelfID:     ob.config.SelfID,
				Type:       libob.EventTypeMessage,
				DetailType: "private",
			},
			UserID:  ob.config.UserID,
			Message: libob.Message{libob.TextSegment(text)},
		})
	}
}
