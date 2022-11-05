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

type responseMsgCode struct {
	Code int `json:"code"`
}

type responseliveness struct {
	Liveness float32
}

type yesterday struct {
	Sum int
}

const (
	FISHPI   = "https://fishpi.cn"
	APIKEY   = "68931c89cc2ecbb1448b2f7edf9b484f8ad006b6dc4fd178d4255821e76c30d78c8b6cc673f15623da1ae4acbda58064a9c6fdca49958af35bfe9abf62f1e8058756d0a6f9740dd38670e7ebe5d69323c2a3296589fd84feefd51e2cd90938bc"
	USERNAME = "bulabula"
)

type messageType map[string]func(message *JSON)

var (
	SendMsg, help           string
	rockMod, heartMod, open bool
	packageContent          getRedpacketContent
	sendResponseContent     responseMsgCode
	liveness                responseliveness
	yesterdayPonit          yesterday
	helpInfo                = []string{
		"*****************************************\n",
		"> #help\n",
		"> 查看帮助文档\n",
		"> 命令均以#号开头目前只支持列出的命令\n",
		"> #rockmod",
		"> 开启抢猜拳红包模式\n",
		"> 提高一捏捏的概率\n",
		"> #heartmod",
		"> 开启抢心跳红包模式\n",
		"> 提高抢到积分的概率以及规避扣积分的概率\n",
		">*****************************************\n",
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
	command = map[string]func(){
		"#rockmod": func() {
			WssSetRockMod()
		},
		"#heartmod": func() {
			WssSetHeartMod()
		},
		"#getpoint": func() {
			WssGetYesterdayPoint()
		},
		"#help": func() {
			WssPrintMsg("Fish机器人", USERNAME, "命令", help)
		},
	}
	reg = []string{
		`^>(.*)(\[.*\]\(.*\)){1,}`,
		`<.*>`,
	}
	client = &http.Client{}
)
