package main

import (
	"encoding/json"
	"strings"
	"time"

	libob "github.com/botuniverse/go-libonebot"
	log "github.com/sirupsen/logrus"
)

type OneBotDummy struct {
	*libob.OneBot
	config *Config
}

type Config struct {
	libob.Config
	UserID string
}

func (ob *OneBotDummy) handleGetSelfInfo(w libob.ResponseWriter, r *libob.Request) {
	w.WriteData(map[string]interface{}{
		"user_id": ob.config.UserID,
	})
}

func main() {
	configString := `
	{
		"Heartbeat": {
			"Enabled": false,
			"Interval": 10
		},
		"Auth": {
			"AccessToken": "abc"
		},
		"CommMethods": {
			"HTTP1": [
				{
					"Host": "127.0.0.1",
					"Port": 5700
				},
				{
					"Host": "127.0.0.1",
					"Port": 5701
				}
			],
			"HTTPWebhook1": [
				{
					"URL": "http://127.0.0.1:8080/"
				}
			],
			"WS": [
				{
					"Host": "127.0.0.1",
					"Port": 6700
				}
			],
			"WSReverse": [
				{
					"URL": "ws://127.0.0.1:8080/ws"
				}
			]
		},
		"UserID": "123"
	}
	`

	config := &Config{}
	json.NewDecoder(strings.NewReader(configString)).Decode(config)

	obdummy := &OneBotDummy{
		OneBot: libob.NewOneBot("dummy", &config.Config),
		config: config,
	}
	obdummy.Logger.SetLevel(log.DebugLevel)

	mux := libob.NewActionMux()
	mux.HandleFunc(libob.ActionGetSelfInfo, obdummy.handleGetSelfInfo)
	mux.HandleFunc(libob.ActionGetVersion, func(w libob.ResponseWriter, r *libob.Request) {
		w.WriteData(map[string]string{
			"version":         "1.0.0",
			"onebot_standard": "v12",
		})
	})
	mux.HandleFunc(libob.ActionSendMessage, func(w libob.ResponseWriter, r *libob.Request) {
		p := libob.NewParamGetter(r.Params, w)
		userID, ok := p.GetString("user_id")
		if !ok {
			return
		}
		msg, ok := p.GetMessage("message")
		if !ok {
			return
		}
		log.Debugf("Send message: %#v, to %v", msg, userID)
		w.WriteData(msg)
	})
	mux.HandleFuncExtended("do_something", func(w libob.ResponseWriter, r *libob.Request) {
	})

	obdummy.Handle(mux)

	go func() {
		for {
			obdummy.Push(
				&libob.MessageEvent{
					Event: libob.Event{
						SelfID:     "123",
						Type:       libob.EventTypeMessage,
						DetailType: "private",
					},
					UserID:  "234",
					Message: libob.Message{libob.TextSegment("hello")},
				},
			)
			time.Sleep(time.Duration(3) * time.Second)
		}
	}()

	// go func() {
	// 	time.Sleep(time.Duration(5) * time.Second)
	// 	obdummy.Shutdown()
	// }()

	obdummy.Run()
}
