package main

import (
	"encoding/json"
	"fmt"
	"github.com/getlantern/systray"
	"github.com/magblank/bbreply/bilibili"
	"github.com/skratchdot/open-golang/open"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ReplyMsg struct {
	ID      int      `json:"ID"`
	Keys    []string `json:"Keys"`    // 关键词列表
	Content string   `json:"Content"` // 回复内容
}

type User struct {
	MID              json.Number         `json:"MID"`
	BName            string              `json:"BName"`
	Cookie           string              `json:"Cookie"`
	ReplyMsgList     []ReplyMsg          `json:"ReplyMsgList"`     // 关注的粉丝回复内容列表
	ReplyMsgUnFollow ReplyMsg            `json:"ReplyMsgUnFollow"` // 没关注的粉丝回复内容
	MidFans          map[string]struct{} `json:"MidFans"`          // 粉丝mid列表
}

type Config struct {
	UserInfo  User `json:"UserInfo"`
	Heartbeat int  `json:"Heartbeat"` // 心跳
}

// SaveConfig 保存配置文件
func (c Config) SaveConfig() error {
	// 将配置数据转换为 JSON 格式
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Printf("Json转换失败：%v\n", err)
		return err
	}
	// 写入配置文件
	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		log.Printf("配置写入失败：%v", err)
		return err
	}
	return err
}

var (
	config         Config
	configFilePath string
)

// SyncFans 同步粉丝数量
func SyncFans(c *bilibili.Client) {
	//判断当前粉丝数与实际粉丝数
	follower, err := c.GetUserFollower(string(config.UserInfo.MID))
	if err != nil {
		log.Fatalf("获取粉丝数失败：%v", err)
	}
	remoteFansMid := follower.Follower
	if len(config.UserInfo.MidFans) >= remoteFansMid {
		// 数据一致
		return
	} else if len(config.UserInfo.MidFans) > remoteFansMid {
		//TODO 本地粉丝数量大于实际粉丝数量
		// 此功能实现方式因为没有确切条件实现所以实现起来很繁琐
		// 1、每次都去获取粉丝数，不现实
		// 2、、、、

	}

	// 取总页数,向上取整
	nps := math.Ceil(float64(remoteFansMid) / 50.00)
	for i := 1; i <= int(nps); i++ {
		fans, err := c.GetFollowerFans(string(config.UserInfo.MID), strconv.Itoa(i), "50", "desc")
		// 防止过快,暂停2秒
		time.Sleep(2 * time.Second)
		if err != nil {
			log.Printf("获取粉丝数失败：%v", err)
		}
		log.Printf("获取%d个粉丝存入本地\n", len(fans.List))
		for _, f := range fans.List {
			config.UserInfo.MidFans[string(f.Mid)] = struct{}{}
		}
		err = config.SaveConfig()
		log.Printf("同步粉丝数量成功：本地粉丝数量%d--b站当前粉丝数：%d", len(config.UserInfo.MidFans), fans.Total)
		if err != nil {
			log.Printf("同步粉丝量时保存失败：%v", err)
		}
		if len(config.UserInfo.MidFans) >= fans.Total {
			//数据一致
			log.Printf("同步完成-本地粉丝数量%d--b站当前粉丝数：%d", len(config.UserInfo.MidFans), fans.Total)
			return
		}
	}
}

func LoginWithQr() *bilibili.Client {
	c := bilibili.New()
	if config.UserInfo.Cookie == "" {
		// 扫码登录
		qrCode, err := c.GetQRCode()
		if err != nil {
			log.Fatalf("二维码获取失败：%v", err)
		}
		buf, err := qrCode.Encode()
		if err != nil {
			log.Printf("保存二维码失败：%v", err)
		}
		err = os.WriteFile("./data/qrcode.png", buf, 0644)
		if err != nil {
			log.Printf("保存二维码失败：%v", err)
		}
		// 显示到终端
		qrCode.Print()
		err = c.LoginWithQRCode(qrCode)
		if err != nil {
			log.Fatalf("登录失败：%v", err)
		}

		userInfo, err := c.GetUserInfo()
		if err != nil {
			log.Printf("获取用户信息失败：%v", err)
		} else {
			log.Printf("登录成功")
			log.Printf("UID:%s", userInfo.Mid)
			log.Printf("U P:%s", userInfo.Uname)
			config.UserInfo.MID = userInfo.Mid
			config.UserInfo.BName = userInfo.Uname
			config.UserInfo.Cookie = c.GetCookiesString()
			err := config.SaveConfig()
			if err != nil {
				log.Printf("保存信息失败：%v\n", err)
			}
			go Reply(c)
		}
	} else {
		// cookie login
		c.SetCookiesString(config.UserInfo.Cookie)
		userInfo, err := c.GetUserInfo()
		if err != nil {
			log.Printf("获取用户信息失败：%v", err)
		} else {
			log.Printf("登录成功")
			log.Printf("UID:%s", userInfo.Mid)
			log.Printf("U P:%s", userInfo.Uname)
			config.UserInfo.MID = userInfo.Mid
			config.UserInfo.BName = userInfo.Uname
			err := config.SaveConfig()
			if err != nil {
				log.Printf("保存信息失败：%v\n", err)
			}
			go Reply(c)
		}
	}
	return c
}

// Reply 自动回复函数
func Reply(c *bilibili.Client) {

	//设置循环获取未读信息，判断关键词回复
	for {
		// 心跳
		time.Sleep(time.Duration(config.Heartbeat) * time.Second)
		//log.Printf("心跳!!!")
		unreadMessage, err := c.GetUnreadPrivateMessage()
		if err != nil {
			log.Printf("获取未读私信数失败：%v\n", err)
			continue
		}

		if unreadMessage.UnfollowUnread != 0 || unreadMessage.FollowUnread != 0 {
			newSessionMessages, err := c.GetNewSessionMessages()
			if err != nil {
				log.Printf("获取新的消息记录失败：%v\n", err)
				continue
			}

			for _, session := range newSessionMessages.SessionList {
				// 发送方为自己跳过
				if session.LastMsg.SenderUID == config.UserInfo.MID {
					continue
				}
				content := session.LastMsg.Content

				// 判断是否是关键词
				if msg, ok := containsKey(content, config.UserInfo.ReplyMsgList); ok {
					log.Printf("开始检查前10粉丝数量")
					//SyncFans(c) // 同步粉丝列表
					// 获取前10粉丝来判断，如果不是则判断全局粉丝（以前关注过但取消了也算粉丝）
					if ok := CheckFontTenFans(c, string(session.LastMsg.SenderUID)); ok {
						// 是关注着且关键词正确
						n, s, err := c.SendPrivateMessageText(session.LastMsg.ReceiverID, session.LastMsg.SenderUID, msg)
						time.Sleep(1 * time.Second) // 发送后延时1秒钟
						if err != nil {
							log.Printf("发送消息失败%v\n", err)
							continue
						}
						log.Printf("%d---%s已使用{{%s}}回复发送方消息：%v\n", n, s, msg, content)
					} else if ok := CheckInMidFans(string(session.LastMsg.SenderUID)); ok {
						// 是关注着且关键词正确
						n, s, err := c.SendPrivateMessageText(session.LastMsg.ReceiverID, session.LastMsg.SenderUID, msg)
						time.Sleep(1 * time.Second) // 发送后延时1秒钟
						if err != nil {
							log.Printf("发送消息失败：%v\n", err)
							continue
						}
						log.Printf("%d---%s已使用{{%s}}回复发送方消息：%v\n", n, s, msg, content)
					} else {
						// 关键词正确，回复未关注着信息
						n, s, err := c.SendPrivateMessageText(session.LastMsg.ReceiverID, session.LastMsg.SenderUID, config.UserInfo.ReplyMsgUnFollow.Content)
						time.Sleep(1 * time.Second) // 发送后延时1秒钟
						if err != nil {
							log.Printf("发送消息失败：%v\n", err)
							continue
						}
						log.Printf("%d----%s已回复发送方消息：%v\n", n, s, content)
					}
				}
			}
		}
	}
}

func CheckFontTenFans(c *bilibili.Client, mid string) bool {
	fans, err := c.GetFollowerFans(string(config.UserInfo.MID), "1", "10", "desc")
	if err != nil {
		log.Printf("获取粉丝数失败：%v", err)
	}
	var data = make(map[string]struct{}, 10)
	for _, f := range fans.List {
		data[string(f.Mid)] = struct{}{}
		config.UserInfo.MidFans[string(f.Mid)] = struct{}{}
	}
	err = config.SaveConfig()
	log.Printf("同步粉丝数量成功：本地粉丝数量%d--b站当前粉丝数：%d", len(config.UserInfo.MidFans), fans.Total)
	if err != nil {
		log.Printf("同步粉丝量时保存失败：%v", err)
	}
	if _, ok := data[mid]; ok {
		return true
	}
	return false
}

// CheckInMidFans 判断发送方是否是粉丝，是->true
func CheckInMidFans(mid string) bool {
	// 表示禁用此功能，直接返回true
	if config.UserInfo.ReplyMsgUnFollow.Content == "" {
		return true
	}
	if _, ok := config.UserInfo.MidFans[mid]; ok {
		return true
	}
	return false
}

// containsKey 判断文本中是否存在给定的关键词,并返回内容
func containsKey(text string, replyMsgList []ReplyMsg) (string, bool) {
	for _, replyMsg := range replyMsgList {
		for _, key := range replyMsg.Keys {
			if strings.Contains(text, key) {
				return replyMsg.Content, true
			}
		}
	}
	return "", false
}

func onReady() {
	systray.SetIcon(Data)
	systray.SetTitle("BBReply")
	systray.SetTooltip("BBReply")

	mOpenOperate := systray.AddMenuItem("打开页面", "打开页面")
	mShowConsole := systray.AddMenuItem("显示控制台", "显示控制台")
	mHideConsole := systray.AddMenuItem("隐藏控制台", "隐藏控制台")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("退出", "退出")

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	// https://docs.microsoft.com/en-us/windows/console/getconsolewindow
	getConsoleWindows := kernel32.NewProc("GetConsoleWindow")
	// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-showwindowasync
	showWindowAsync := user32.NewProc("ShowWindowAsync")
	consoleHandle, r2, err := getConsoleWindows.Call()
	if consoleHandle == 0 {
		fmt.Println("Error call GetConsoleWindow: ", consoleHandle, r2, err)
	}

	go func() {
		for {
			select {
			case <-mOpenOperate.ClickedCh:
				//打开操作界面
				open.Run("http://127.0.0.1:9558")
			case <-mShowConsole.ClickedCh:
				mShowConsole.Disable()
				mHideConsole.Enable()
				r1, r2, err := showWindowAsync.Call(consoleHandle, 5)
				if r1 != 1 {
					log.Println("Error call ShowWindow @SW_SHOW: ", r1, r2, err)
				}
			case <-mHideConsole.ClickedCh:
				mHideConsole.Disable()
				mShowConsole.Enable()
				r1, r2, err := showWindowAsync.Call(consoleHandle, 0)
				if r1 != 1 {
					log.Println("Error call HideWindow @SW_SHOW: ", r1, r2, err)
				}
			case <-mExit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// clean up here
	log.Printf("Exit ...")
}

func init() {
	// 获取程序运行时目录
	runTimeDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		os.Exit(1)
	}

	// 拼接data文件夹的路径
	dataDir := filepath.Join(runTimeDir, "data")
	// 检查data文件夹是否存在，如果不存在则创建
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err := os.MkdirAll(dataDir, 0755) // 0755是目录权限，你可以根据需要修改
		if err != nil {
			log.Printf("创建data文件夹失败:%v\n", err)
			return
		}
	}

	// 拼接err.log文件的路径
	logFilePath := filepath.Join(dataDir, "err.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("%v", err)
	}
	defer logFile.Close()
	// 设置日志的输出目的地为文件
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	// 拼接config.json文件的路径
	configFilePath = filepath.Join(dataDir, "config.json")
	// 检查config.json文件是否存在，如果不存在则创建
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		file, err := os.Create(configFilePath)
		if err != nil {
			log.Printf("创建config.json文件失败:%v\n", err)
			return
		}
		defer file.Close()
	} else {
		// 解析config.json文件
		fileData, err := os.ReadFile(configFilePath)
		if err != nil {
			log.Printf("读取config.json文件失败:%v\n", err)
		}

		if len(fileData) > 0 {
			err = json.Unmarshal(fileData, &config)
			if err != nil {
				log.Fatalf("解析配置文件失败，建议删除config.json:%v\n", err)
			}
		}

		if config.UserInfo.Cookie != "" {

		}
	}
}

func main() {

	// 默认心跳为10s
	if config.Heartbeat == 0 {
		config.Heartbeat = 10
	}
	// 申请空间
	if len(config.UserInfo.MidFans) == 0 {
		config.UserInfo.MidFans = make(map[string]struct{}, 1000)
	}
	// 登录
	c := LoginWithQr()

	// 将粉丝mid保存到本地
	go SyncFans(c)

	// 启动web服务
	go StartServer()

	systray.Run(onReady, onExit)

}
