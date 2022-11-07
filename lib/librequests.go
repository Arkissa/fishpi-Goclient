package lib

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
)

type fishpiUserProperty struct {
	ApiKey   string
	Origin   string
	SendMsg  string
	UserName string `json:"username"`
	Password string `json:"password"`
	Exit     chan os.Signal
	message  JSON
}

func NewFishpi() (*fishpiUserProperty, error) {
	fish := &fishpiUserProperty{
		Origin: "https://fishpi.cn",
		Exit:   make(chan os.Signal),
	}
	f, err := os.Open("config.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	j := json.NewDecoder(f)
	err = j.Decode(fish)
	if err != nil {
		return nil, err
	}
	err = fish.WssLogin()
	if err != nil {
		return nil, err
	}
	fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", "登陆成功")
	return fish, nil
}

func (fish *fishpiUserProperty) WssLogin() error {
	var apikey apiKeyContent
	fish.md5String(&fish.Password)
	k := make(map[string]string)
	k["nameOrEmail"] = fish.UserName
	k["userPassword"] = fish.Password
	k["mfaCode"] = ""
	str := k["mfaCode"]
	fmt.Printf("%s\n>", "两步认证一次性密码（如未设置请输入0）")
	if _, err := fmt.Scanln(&str); err != nil {
		return err
	}
	if str == "0" {
		str = ""
	}
	k["mfaCode"] = strings.Split(str, "\n")[0]
	responseBody, err := Requests("POST", "https://fishpi.cn/api/getKey", k)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(responseBody, &apikey)
	if apikey.Code != 0 {
		return errors.New(apikey.Msg)
	}
	fish.ApiKey = apikey.Key
	return nil
}

func (fish *fishpiUserProperty) md5String(msg *string) {
	s := *msg
	m := md5.New()
	m.Write(([]byte)(s))
	*msg = hex.EncodeToString(m.Sum(nil))
}

func (fish *fishpiUserProperty) WssLink() {
	url := "wss://fishpi.cn/chat-room-channel?apiKey=" + fish.ApiKey

	signal.Notify(fish.Exit, os.Interrupt, syscall.SIGKILL)
	go func() {
		for {
			rand.Seed(time.Now().Unix())
			time.Sleep(time.Duration(rand.Intn(30)+30) * time.Second)
			fish.WssGetLiveness()
		}
	}()

	go func(exit chan os.Signal) {
		<-exit
		fmt.Println("\nCtrl + c program exit……")
		os.Exit(0)
	}(fish.Exit)

	mt := messageType{
		"msg": func(message *JSON) {
			var md string
			if strings.Contains((*message).Content, "redPacket") {
				fish.WssOpenRedPacket(message)
			} else {
				err := msgHandle(&message.Md, &md, reg)
				if err != nil {
					log.Println("message handle err: ", err)
					return
				}
				fish.WssPrintMsg(message.UserNickname, message.UserName, "消息", md)
			}
		},

		"online": func(message *JSON) {
			// ...
		},
		"redPacketStatus": func(message *JSON) {
			// ...
		},
		"discussChanged": func(message *JSON) {
			// ...
		},
		"revoke": func(message *JSON) {
			// ...
		},
	}

	ws, err := websocket.Dial(url, "", fish.Origin)
	if err != nil {
		fmt.Println("link error: ", err)
	}

	defer ws.Close()
	for {
		if err = websocket.JSON.Receive(ws, &fish.message); err != nil {
			log.Println("websocket json Unmarshal err: ", err)
			return
		}
		go mt[fish.message.Type](&fish.message)
	}
}

func (fish *fishpiUserProperty) WssOpenRedPacket(msg *JSON) {
	var (
		n          int
		m          string
		open       bool
		msgContent redContent
	)
	openRedPacke["apiKey"] = fish.ApiKey
	openRedPacke["oId"] = msg.OID
	json.Unmarshal([]byte(msg.Content), &msgContent)
	go fish.WssPrintMsg(msg.UserNickname, msg.UserName, redType[msgContent.Type], msgContent.Msg)
	if msgContent.Type == "rockPaperScissors" && rockMod {
		rand.Seed(time.Now().Unix())
		n = rand.Intn(2)
		n = rand.Intn(2)
		openRedPacke["gesture"] = fmt.Sprintf("%d", n)
	} else {
		return
	}

	if msgContent.Type == "heartbeat" && heartMod {
		open = fish.redPacketStatus(msg, &msgContent)
		if !open {
			return
		}
	} else {
		return
	}

	start := time.Now().Unix()
	if !open {
		for k := (int64)(0); k < 3; k = (time.Now().Unix() - start) {
		}
	}
	responseBody, err := Requests("POST", "https://fishpi.cn/chat-room/red-packet/open", openRedPacke)
	if err != nil {
		log.Println("requests set err: ", err)
		return
	}
	if err = json.Unmarshal(responseBody, &packageContent); err != nil {
		log.Println("send message response json unmarshal err: ", err)
	}
	for _, value := range packageContent.Who {
		if value.UserMoney >= 0 {
			m = fmt.Sprintf("抢到%d积分！！", value.UserMoney)
		} else {
			m = fmt.Sprintf("失去了%d积分……", value.UserMoney)
		}
		go fish.WssPrintMsg("红包机器人", value.UserName, "打开了红包", m)
	}
}

func (fish *fishpiUserProperty) redPacketStatus(msg *JSON, msgContent *redContent) bool {
	var (
		p            float32
		now          int64
		responseBody []byte
		err          error
		i            int
		j            = 1
		t            = time.Now().Unix()
		old          oldMsgContent
	)

	for {
		p = float32(msgContent.Got) / (float32)(msgContent.Count) * 100.00
		if i > 24 {
			i = 0
			j += 1
		}

		responseBody, err = Requests("GET", fmt.Sprintf("https://fishpi.cn/chat-room/more?apiKey=%s&page=%d", fish.ApiKey, j), nil)
		if err != nil {
			fish.WssPrintMsg("红包机器人", msg.UserName, "打开红包", err.Error())
			return false
		}
		_ = json.Unmarshal(responseBody, &old)
		err = json.Unmarshal(([]byte)(old.Data[i].Content), &msgContent)
		if err != nil {
			i++
			continue
		}
		for _, value := range msgContent.Who {
			if value.UserMoney >= 1 {
				fish.WssPrintMsg("红包机器人", value.UserName, "打开了红包",
					fmt.Sprintf("%s抢到了%d积分?!不好有诈！快跑！！", value.UserName, value.UserMoney))
				return false
			}
		}
		if p > 70.00 {
			fish.WssPrintMsg("红包机器人", msg.UserName, "冲！！！", fmt.Sprintf("我超！有%.2f%%几率了！我冲了！！！", p))
			return true
		}
		now = time.Now().Unix() - t
		if now >= (int64)(5) {
			fish.WssPrintMsg("红包机器人", msg.UserName, "冲！！！", "时间到了我憋不住了！我冲了！！！")
			return true
		}
		fish.WssPrintMsg("红包机器人", msg.UserName, "等待",
			fmt.Sprintf("不急现在拿到红包的几率只有%.2f%%已经过了%d秒了再等等！！", p, now))
	}

}

func WssOnline(mes *JSON) {

}

func (fish *fishpiUserProperty) WssSendMsg() {
	sendMessage["apiKey"] = fish.ApiKey
	sendMessage["content"] = fmt.Sprintf("%s\n>————————*Go client [%.2f%%]*", fish.SendMsg, liveness.Liveness)
	responseBody, err := Requests("POST", "https://fishpi.cn/chat-room/send", sendMessage)
	if err != nil {
		log.Println("requests set err: ", err)
		return
	}
	err = json.Unmarshal(responseBody, &sendResponseContent)
	if err != nil {
		log.Println("send message response json unmarshal err: ", err)
	}
	if sendResponseContent.Code != 0 {
		log.Println("send message fail  response code is: ", sendResponseContent.Code)
	}
}

func (fish *fishpiUserProperty) WssGetLiveness() {
	responseBody, err := Requests("GET", "https://fishpi.cn/user/liveness?apiKey="+fish.ApiKey, nil)
	if err != nil {
		log.Println("requests set err: ", err)
		return
	}
	_ = json.Unmarshal(responseBody, &liveness)
}

func msgHandle(msg, md *string, reg []string) (err error) {
	var re *regexp.Regexp
	for _, r := range reg {
		re, err = regexp.Compile(r)
		if err != nil {
			return err
		}
		*msg = re.ReplaceAllString(*msg, "")
	}

	for i, char := range *msg {
		if i+1 == len(*msg) {
			continue
		}
		if char == '\n' {
			if (*msg)[i+1] == '\n' {
				continue
			}
		}
		*md += (string)(char)
	}
	return nil
}

func Requests(mode, url string, content map[string]string) (body []byte, err error) {
	var (
		request  *http.Request
		response *http.Response
	)

	requestBody, err := setRequestBody(content)
	if err != nil {
		return nil, err
	}

	request, err = http.NewRequest(mode, url, requestBody)
	if err != nil {
		log.Println("set New request header and body err: ", err)
		return nil, err
	}

	request.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36")
	if mode == "POST" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err = client.Do(request)
	if err != nil {
		log.Println("send request err: ", err)
		return nil, err
	}
	defer func() {
		err = response.Body.Close()
	}()
	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("reader response body err: ", err)
		return nil, err
	}
	return body, nil
}

func setRequestBody(content map[string]string) (io.Reader, error) {
	marshalBody, err := json.Marshal(content)
	if err != nil {
		log.Println("request body marshal err: ", err)
		return nil, err
	}

	return bytes.NewReader(marshalBody), nil
}

func (fish *fishpiUserProperty) WssClient() {
	var c []string
	for _, n := range helpInfo {
		help += n
	}
	fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", help)
	go fish.WssGetLiveness()
	for {
		b := bufio.NewReaderSize(os.Stdin, 512)
		if s, err := b.ReadString('\n'); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			s = strings.Split(s, "\n")[0]
			if strings.HasPrefix(s, "#") {
				c = strings.Split(s, " ")
				switch c[0] {
				case "#rockmod":
					fish.WssSetRockMod()
				case "#heartmod":
					fish.WssSetHeartMod()
				case "#getpoint":
					fish.WssGetYesterdayPoint()
				case "#help":
					fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", help)
				default:
					fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", "没有此命令"+c[0])
				}
			} else {
				fish.SendMsg = s
				fish.WssSendMsg()
			}
		}
	}
}

func (fish *fishpiUserProperty) WssSetRockMod() {
	s := ""
	if !rockMod {
		rockMod = true
		s = "开启"
	} else {
		rockMod = false
		s = "关闭"
	}
	fish.WssPrintMsg("Fish机器人", "rockMod", "命令", "rockMod"+s)
}

func (fish *fishpiUserProperty) WssSetHeartMod() {
	s := ""
	if !heartMod {
		heartMod = true
		s = "开启"
	} else {
		heartMod = false
		s = "关闭"
	}
	fish.WssPrintMsg("Fish机器人", "heartMod", "命令", "heartMod"+s)
}

func (fish *fishpiUserProperty) WssGetYesterdayPoint() {
	responseBody, err := Requests("GET", "https://fishpi.cn/activity/yesterday-liveness-reward-api?apiKey="+fish.ApiKey, nil)
	s := ""
	if err != nil {
		log.Println("get yesterday point err:", err)
		return
	}
	err = json.Unmarshal(responseBody, &yesterdayPonit)
	if err != nil {
		log.Println(err)
		return
	}
	switch yesterdayPonit.Sum {
	case -1:
		s = "已经领取过积分了"
	default:
		s = fmt.Sprintf("抢到到%d积分", yesterdayPonit.Sum)
	}
	fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", s)
}

func (fish *fishpiUserProperty) WssPrintMsg(userNickname, userName, info, md string) {
	fmt.Printf("%s(%s)(%s):\n>%s\n\n", userNickname, userName, info, md)
	fmt.Print(">")
}
