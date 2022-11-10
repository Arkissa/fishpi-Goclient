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
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/websocket"
)

type fishpiUserProperty struct {
	ApiKey         string
	Origin         string
	SendMsg        string
	ImageUrl       string
	UserName       string `json:"username"`
	Password       string `json:"password"`
	Exit           chan os.Signal
	message        JSON
	RequestContent struct {
		content     map[string]string
		imageBuffer io.Reader
		contentType string
	}
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
	var (
		apikey apiKeyContent
	)
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
	fish.RequestContent.content = k
	fish.RequestContent.imageBuffer = nil
	responseBody, err := fish.Requests("POST", "https://fishpi.cn/api/getKey")
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
			if strings.Contains((*message).Content, "redPacket") {
				fish.WssOpenRedPacket(message)
			} else {
				md, err := msgHandle(&message.Md, reg)
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
	}

	if msgContent.Type == "heartbeat" && heartMod {
		open = fish.redPacketStatus(msg, &msgContent)
		if !open {
			return
		}
	}

	start := time.Now().Unix()
	if !open {
		for k := (int64)(0); k < 3; k = (time.Now().Unix() - start) {
		}
	}
	fish.RequestContent.content = openRedPacke
	fish.RequestContent.imageBuffer = nil
	responseBody, err := fish.Requests("POST", "https://fishpi.cn/chat-room/red-packet/open")
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

		responseBody, err = fish.Requests("GET", fmt.Sprintf("https://fishpi.cn/chat-room/more?apiKey=%s&page=%d", fish.ApiKey, j))
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

func (fish *fishpiUserProperty) sendMsg(content string) {
	sendMessage["apiKey"] = fish.ApiKey
	sendMessage["content"] = content
	fish.RequestContent.content = sendMessage
	fish.RequestContent.imageBuffer = nil
	responseBody, err := fish.Requests("POST", "https://fishpi.cn/chat-room/send")
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
	responseBody, err := fish.Requests("GET", "https://fishpi.cn/user/liveness?apiKey="+fish.ApiKey)
	if err != nil {
		log.Println("requests set err: ", err)
		return
	}
	_ = json.Unmarshal(responseBody, &liveness)
}

func msgHandle(msg *string, reg []string) (md string, err error) {
	var (
		re  *regexp.Regexp
		tmp = *msg
	)
	for _, r := range reg {
		re, err = regexp.Compile(r)
		if err != nil {
			return "", err
		}
		tmp = re.ReplaceAllString(tmp, "")
	}
	for i, char := range tmp {
		if char == '\n' && i+1 != len(tmp) {
			if tmp[i+1] == '\n' {
				continue
			}
		}
		md += (string)(char)
	}
	return md, nil
}

func (fish *fishpiUserProperty) Requests(mode, url string) (body []byte, err error) {
	var (
		request     *http.Request
		response    *http.Response
		requestBody io.Reader
	)

	if fish.RequestContent.imageBuffer != nil {
		requestBody = fish.RequestContent.imageBuffer
	} else {
		requestBody, err = setRequestBody(fish.RequestContent.content)
		if err != nil {
			return nil, err
		}
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
	if fish.RequestContent.imageBuffer != nil {
		request.Header.Set("Content-Type", fish.RequestContent.contentType)
	}

	response, err = client.Do(request)
	if err != nil {
		log.Println("send request err: ", err)
		return nil, err
	}
	defer response.Body.Close()

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

func setImageRequestBody(path string) (io.Reader, string, error) {
	var (
		suffix  = [4]string{"png", "jpg", "gif", "bmp"}
		srcFile io.Reader
		out     []byte
		err     error
	)

	buff := new(bytes.Buffer)
	writer := multipart.NewWriter(buff)
	fromFile, err := writer.CreateFormFile("file[]", "tmp.png")
	if err != nil {
		return nil, "", err
	}

	if path == "" {
		for i := 0; i < 4; i++ {
			exe := exec.Command("/bin/bash", "-c", "xclip -sel -c -t image/"+suffix[i]+" -o")
			out, err = exe.CombinedOutput()
			if err != nil && i == len(suffix)-1 {
				return nil, "", errors.New("剪切板里数据未知类型")
			}
			break
		}
		reader := bytes.NewReader(out)
		srcFile = reader
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		srcFile = file
		defer file.Close()
	}

	_, err = io.Copy(fromFile, srcFile)
	if err != nil {
		return nil, "", err
	}
	contentType := writer.FormDataContentType()
	writer.Close()
	return buff, contentType, nil
}

func (fish *fishpiUserProperty) WssSendImage(path string) {
	var (
		contentType  string
		buff         io.Reader
		err          error
		responseBody []byte
		image        imageUpload
	)
	buff, contentType, err = setImageRequestBody(path)
	if err != nil {
		fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", err.Error())
		return
	}
	fish.RequestContent.content = nil
	fish.RequestContent.imageBuffer = buff
	fish.RequestContent.contentType = contentType
	responseBody, err = fish.Requests("POST", "https://fishpi.cn/upload")
	if err != nil {
		fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", err.Error())
		return
	}
	json.Unmarshal(responseBody, &image)
	fish.ImageUrl = image.Data.SuccMap.TmpPng
	fish.WssSendMsg("img")
}

func (fish *fishpiUserProperty) WssSendMsg(mold string) {
	var content string
	switch mold {
	case "msg":
		content = fmt.Sprintf("%s\n>————*Go client [%.2f%%]*", fish.SendMsg, liveness.Liveness)
	case "img":
		content = fmt.Sprintf("![images](%s)\n>————*Go client [%.2f%%]*", fish.ImageUrl, liveness.Liveness)
	}
	fish.sendMsg(content)
}

func (fish *fishpiUserProperty) WssClient() {
	var (
		c []string
		s string
	)
	for _, n := range helpInfo {
		help += n
	}
	fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", help)
	go fish.WssGetLiveness()
	b := bufio.NewScanner(os.Stdin)
	for b.Scan() {
		s = strings.Split(b.Text(), "\n")[0]
		if strings.HasPrefix(s, "#") {
			c = strings.Split(s, " ")
			switch c[0] {
			case "#rockmod":
				fish.WssSetRockMod()
			case "#heartmod":
				fish.WssSetHeartMod()
			case "#getpoint":
				fish.WssGetYesterdayPoint()
			case "#img":
				if len(c) < 2 {
					fish.WssSendImage("")
				} else {
					fish.WssSendImage(c[1])
				}
			case "#help":
				fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", help)
			default:
				fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", "没有此命令"+c[0])
			}
		} else {
			fish.SendMsg = s
			fish.WssSendMsg("msg")
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
	responseBody, err := fish.Requests("GET", "https://fishpi.cn/activity/yesterday-liveness-reward-api?apiKey="+fish.ApiKey)
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
		s = fmt.Sprintf("获到%d积分", yesterdayPonit.Sum)
	}
	fish.WssPrintMsg("Fish机器人", fish.UserName, "命令", s)
}

func (fish *fishpiUserProperty) WssPrintMsg(userNickname, userName, info, md string) {
	fmt.Printf("%s(%s)(%s):\n>%s\n\n", userNickname, userName, info, md)
	fmt.Printf(">")
}
