package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	libob "github.com/botuniverse/go-libonebot"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type OneBotREPL struct {
	*libob.OneBot // 嵌入 OneBot 对象
	config        *REPLConfig
	lastMessageID uint64
}

type Config struct {
	OneBot libob.Config `mapstructure:",squash"` // 嵌入 LibOneBot 配置
	REPL   REPLConfig
}

type REPLConfig struct {
	SelfID string `mapstructure:"self_id"`
	UserID string `mapstructure:"user_id"`
}

const defaultConfigString = `
[heartbeat]
enabled = true
interval = 10

[auth]
access_token = ""

[repl]
self_id = "bot"
user_id = "user"
`

func loadConfig() *Config {
	// 使用 viper 库加载配置
	v := viper.New()
	v.SetConfigType("toml")
	v.ReadConfig(strings.NewReader(defaultConfigString)) // 加载默认配置
	v.SetConfigFile("config.toml")
	err := v.MergeInConfig() // 合并配置文件内容
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
	// 加载配置
	config := loadConfig()

	// 创建 OneBot 实例
	ob := &OneBotREPL{
		OneBot:        libob.NewOneBot("repl", &config.OneBot),
		config:        &config.REPL,
		lastMessageID: 0,
	}

	// 修改日志配置
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	ob.Logger.SetOutput(logFile)
	ob.Logger.SetLevel(logrus.InfoLevel)

	// 通过 ActionMux 注册动作处理函数，该 mux 变量可在多个 OneBot 实例复用
	mux := libob.NewActionMux()
	// 注册 get_version 动作处理函数
	mux.HandleFunc(libob.ActionGetVersion, func(w libob.ResponseWriter, r *libob.Request) {
		// 返回一个映射类型的数据（序列化为 JSON 对象或 MsgPack 映射）
		w.WriteData(map[string]string{
			"version": "1.0.0",
		})
	})
	// 注册 get_self_id 动作处理函数
	mux.HandleFunc(libob.ActionGetSelfInfo, func(w libob.ResponseWriter, r *libob.Request) {
		w.WriteData(map[string]interface{}{
			"user_id": ob.config.SelfID, // 返回配置中指定的 self_id
		})
	})
	// 注册 send_message 动作处理函数
	mux.HandleFunc(libob.ActionSendMessage, func(w libob.ResponseWriter, r *libob.Request) {
		// 创建 ParamGetter 来获取参数，也可以直接用 r.Params.GetXxx
		p := libob.NewParamGetter(w, r)
		userID, ok := p.GetString("user_id")
		if !ok {
			return
		}
		if userID != ob.config.UserID {
			// user_id 不匹配，返回 RetCodeLogicError
			w.WriteFailed(libob.RetCodeLogicError, fmt.Errorf("无法发送给用户 `%v`", userID))
			return
		}
		msg, ok := p.GetMessage("message")
		if !ok {
			return
		}
		fmt.Println(msg.ExtractText()) // 提取消息中的纯文本并打印在控制台
		// 返回消息 ID
		w.WriteData(map[string]interface{}{
			"message_id": fmt.Sprint(atomic.AddUint64(&ob.lastMessageID, 1)),
		})
	})
	// 注册 repl_test 扩展动作处理函数
	mux.HandleFunc("repl_test", func(w libob.ResponseWriter, r *libob.Request) {
		w.WriteData("It works!") // 返回一个字符串
	})

	ob.Handle(mux)
	go ob.Run() // 启动 OneBot 实例

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("请开始对话 (输入 exit 退出):")
	// 循环读取命令行输入
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "exit" {
			ob.Shutdown()
			break
		}
		// 构造 OneBot 私聊消息事件并通过 OneBot 对象推送到机器人业务端
		go ob.Push(&libob.PrivateMessageEvent{
			MessageEvent: libob.MessageEvent{
				Event: libob.Event{
					SelfID:     ob.config.SelfID,
					Type:       libob.EventTypeMessage,
					DetailType: "private",
				},
				Message: libob.Message{libob.TextSegment(text)},
			},
			UserID: ob.config.UserID,
		})
	}
}
