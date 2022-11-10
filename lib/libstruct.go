package lib

import (
	"net/http"
)

type oldMsgContent struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
	Data []struct {
		UserNickname string `json:"userNickname"`
		SysMetal     string `json:"sysMetal"`
		OID          string `json:"oId"`
		UserName     string `json:"userName"`
		Content      string `json:"content"`
	} `json:"data"`
}

type mdContent struct {
	Md           string `json:"md"`
	UserNickname string `json:"userNickname"`
	OID          string `json:"oId"`
	UserName     string `json:"userName"`
	Type         string `json:"type"`
	Content      string `json:"content"`
}

type redContent struct {
	Msg      string `json:"msg"`
	Recivers string `json:"recivers"`
	MsgType  string `json:"msgType"`
	Money    int    `json:"money"`
	Count    int    `json:"count"`
	Type     string `json:"type"`
	Got      int    `json:"got"`
	Who      []struct {
		UserMoney int    `json:"userMoney"`
		Time      string `json:"time"`
		Avatar    string `json:"avatar"`
		UserName  string `json:"userName"`
		UserID    string `json:"userId"`
	} `json:"who"`
}

type getRedpacketContent struct {
	Who []struct {
		UserMoney int    `json:"userMoney"`
		Time      string `json:"time"`
		Avatar    string `json:"avatar"`
		UserName  string `json:"userName"`
		UserID    string `json:"userId"`
	} `json:"who"`
}

type imageUpload struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
	Data struct {
		ErrFiles []interface{} `json:"errFiles"`
		SuccMap  struct {
			TmpPng string `json:"tmp.png"`
		} `json:"succMap"`
	} `json:"data"`
}

type apiKeyContent struct {
	Key  string `json:"key"`
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type responseMsgCode struct {
	Code int `json:"code"`
}

type responseliveness struct {
	Liveness float32 `json:"liveness"`
}

type yesterday struct {
	Sum int `json:"sum"`
}

type messageType map[string]func(message *JSON)

type JSON mdContent

var (
	help                string
	rockMod, heartMod   bool
	packageContent      getRedpacketContent
	sendResponseContent responseMsgCode
	liveness            responseliveness
	yesterdayPonit      yesterday
	helpInfo            = []string{
		"------------------------------------------\n",
		"> #help\n",
		"> 查看帮助文档\n",
		"> 命令均以#号开头目前只支持列出的命令\n",
		"> #rockmod\n",
		"> 开启抢猜拳红包模式\n",
		"> 提高一捏捏的概率\n",
		"> #heartmod\n",
		"> 开启抢心跳红包模式\n",
		"> 提高抢到积分的概率以及规避扣积分的概率\n",
		"> #getpoint\n",
		"> 领取昨日活跃奖励\n",
		"> #img\n",
		"> 图形桌面linux需要安装xlicp与剪切板交互\n",
		"> windows只支持路径发送\n",
		"> 路径发送需要#img /xx/xxx/xxx.png格式发送\n",
		"------------------------------------------\n",
		">ctrl + c 退出程序\n",
	}
	redChannel   = make(chan bool)
	sendMessage  = make(map[string]string)
	openRedPacke = make(map[string]string)
	redType      = map[string]string{
		"heartbeat":         "心跳红包",
		"random":            "拼手气红包",
		"average":           "普通红包",
		"specify":           "专属红包",
		"rockPaperScissors": "猜拳红包",
	}
	reg = []string{
		`(?m:^>(.*?)((\[.*?\]\(.*?\))){1,})`,
		`(?m:<.*?>)`,
	}
	client = &http.Client{}
)
