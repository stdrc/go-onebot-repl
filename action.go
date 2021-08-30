package main

import (
	"fmt"
	"sync/atomic"

	libob "github.com/botuniverse/go-libonebot"
)

func (ob *OneBotDummy) handleGetVersion(w libob.ResponseWriter, r *libob.Request) {
	w.WriteData(map[string]string{
		"version": "1.0.0",
	})
}

func (ob *OneBotDummy) handleGetSelfInfo(w libob.ResponseWriter, r *libob.Request) {
	w.WriteData(map[string]interface{}{
		"user_id": ob.config.SelfID,
	})
}

func (ob *OneBotDummy) handleSendMessage(w libob.ResponseWriter, r *libob.Request) {
	p := libob.NewParamGetter(w, r)
	userID, ok := p.GetString("user_id")
	if !ok {
		return
	}
	if userID != ob.config.UserID {
		w.WriteFailed(libob.RetCodeLogicError, fmt.Errorf("无法发送给用户 `%v`", userID))
		return
	}
	msg, ok := p.GetMessage("message")
	if !ok {
		return
	}
	fmt.Println(msg.ExtractText())
	w.WriteData(map[string]interface{}{
		"message_id": fmt.Sprint(atomic.AddUint64(&ob.lastMessageID, 1)),
	})
}
