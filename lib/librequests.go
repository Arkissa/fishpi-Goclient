package lib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type JSON mdContent

func WssLink() {
	origin := FISHPI
	url := "wss://fishpi.cn/chat-room-channel?apiKey=" + APIKEY
	go func() {
		for {
			rand.Seed(time.Now().Unix())
			time.Sleep(time.Duration(rand.Intn(30)+30) * time.Second)
			WssGetLiveness()
		}
	}()
	var message JSON
	mt := messageType{
		"msg": func(message *JSON) {

			if strings.Contains((*message).Content, "redPacket") {
				WssOpenRedPacket(message)
			} else {
				err := msgHandle(&message.Md, reg)
				if err != nil {
					log.Println("message handle err: ", err)
					return
				}
				WssPrintMsg(message.UserNickname, message.UserName, "消息", message.Md)
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

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		fmt.Println("link error: ", err)
	}

	defer ws.Close()
	for {
		if err = websocket.JSON.Receive(ws, &message); err != nil {
			log.Println("websocket json Unmarshal err: ", err)
			return
		}
		go mt[message.Type](&message)
	}
}

func WssOpenRedPacket(msg *JSON) {
	var (
		n          int
		m          string
		msgContent redContent
	)
	open = true
	openRedPacke["apiKey"] = APIKEY
	openRedPacke["oId"] = msg.OID
	json.Unmarshal([]byte(msg.Content), &msgContent)
	if msgContent.Type == "rockPaperScissors" && rockMod {
		rand.Seed(time.Now().Unix())
		n = rand.Intn(2)
		n = rand.Intn(2)
		openRedPacke["gesture"] = fmt.Sprintf("%d", n)
	} else {
		open = false
	}

	if msgContent.Type == "heartbeat" && heartMod {
		if !redPacketStatus(msg, &msgContent) {
			return
		}
	} else {
		open = false
	}

	go WssPrintMsg(msg.UserNickname, msg.UserName, redType[msgContent.Type], msgContent.Msg)

	time.Sleep(3 * time.Second)
	if !open {
		return
	} else {
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
				m = fmt.Sprintf("获取%d积分！！", value.UserMoney)
			} else {
				m = fmt.Sprintf("失去了%d积分……", value.UserMoney)
			}
			go WssPrintMsg("红包机器人", value.UserName, "打开了红包", m)
		}
	}
}

func redPacketStatus(msg *JSON, msgContent *redContent) bool {
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

		responseBody, err = Requests("GET", fmt.Sprintf("https://fishpi.cn/chat-room/more?apiKey=%s&page=%d", APIKEY, j), nil)
		if err != nil {
			WssPrintMsg("红包机器人", msg.UserName, "打开红包", err.Error())
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
				WssPrintMsg("红包机器人", value.UserName, "打开了红包",
					fmt.Sprintf("%s获取了%d积分?!不好有诈！快跑！！", value.UserName, value.UserMoney))
				return false
			}
		}
		if p > 70.00 {
			WssPrintMsg("红包机器人", msg.UserName, "冲！！！", fmt.Sprintf("我超！有%.2f%%几率了！我冲了！！！", p))
			return true
		}
		now = time.Now().Unix() - t
		if now >= (int64)(5) {
			WssPrintMsg("红包机器人", msg.UserName, "冲！！！", "时间到了我憋不住了！我冲了！！！")
			return true
		}
		WssPrintMsg("红包机器人", msg.UserName, "等待",
			fmt.Sprintf("不急现在拿到红包的几率只有%.2f%%已经过了%d秒了再等等！！", p, now))
	}

}

func WssOnline(mes *JSON) {

}

func WssSendMsg() {
	sendMessage["apiKey"] = APIKEY
	sendMessage["content"] = fmt.Sprintf("%s\n>————from Go client [*%.2f%%*]", SendMsg, liveness.Liveness)
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

func WssGetLiveness() {
	responseBody, err := Requests("GET", "https://fishpi.cn/user/liveness?apiKey="+APIKEY, nil)
	if err != nil {
		log.Println("requests set err: ", err)
		return
	}
	_ = json.Unmarshal(responseBody, &liveness)
}

func msgHandle(msg *string, reg []string) (err error) {
	var re *regexp.Regexp
	for _, r := range reg {
		re, err = regexp.Compile(r)
		if err != nil {
			return err
		}
		*msg = re.ReplaceAllString(*msg, "")
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

func WssClient() {
	for _, n := range helpInfo {
		help += n
	}
	for {
		b := bufio.NewReaderSize(os.Stdin, 512)
		if s, err := b.ReadString('\n'); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			s = strings.Split(s, "\n")[0]
			c := strings.Split(s, " ")
			if strings.HasPrefix(c[0], "#") {
				if len(c) > 1 {
					_ = c[1:]
				}
				command[c[0]]()
			} else {
				SendMsg = c[0]
				WssSendMsg()
			}
		}
	}

}

func WssSetRockMod() {
	s := ""
	if !rockMod {
		rockMod = true
		s = "开启"
	} else {
		rockMod = false
		s = "关闭"
	}
	WssPrintMsg("Fish机器人", "rockMod", "命令", "rockMod"+s)
}

func WssSetHeartMod() {
	s := ""
	if !heartMod {
		heartMod = true
		s = "开启"
	} else {
		heartMod = false
		s = "关闭"
	}
	WssPrintMsg("Fish机器人", "heartMod", "命令", "heartMod"+s)
}

func WssGetYesterdayPoint() {
	responseBody, err := Requests("GET", "https://fishpi.cn/api/activity/is-collected-liveness?apiKey="+APIKEY, nil)
	s := ""
	if err != nil {
		log.Println("get yesterday point err:", err)
	}
	_ = json.Unmarshal(responseBody, &yesterdayPonit)
	switch yesterdayPonit.Sum {
	case -1:
		s = "已经领取过积分了"
	default:
		s = fmt.Sprintf("获取到%d积分", yesterdayPonit.Sum)
	}
	WssPrintMsg("Fish机器人", USERNAME, "命令", s)
}

func WssPrintMsg(userNickname, userName, info, md string) {
	fmt.Printf("%s(%s)(%s):\n>%s\n\n", userNickname, userName, info, md)
}
