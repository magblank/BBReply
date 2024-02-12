package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin的实例
	r := gin.Default()

	// 设置静态资源
	r.Static("/assets", "data/dist/assets/")
	r.StaticFile("favicon.ico", "data/dist/favicon.ico")

	//跨域
	r.Use(cors.Default())

	// 路由
	r.GET("/", func(c *gin.Context) {
		c.File("data/dist/index.html")
	})

	api := r.Group("/api")
	api.GET("/index", Index)
	api.POST("/add", Add)
	api.POST("/unfollower", AddUnFollower)
	api.POST("/save", Save)
	api.GET("/del", Del)

	return r
}

func AddUnFollower(c *gin.Context) {
	var reqData struct {
		ReplyMsgUnFollow string `json:"ReplyMsgUnFollow"`
	}
	err := c.ShouldBindJSON(&reqData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.UserInfo.ReplyMsgUnFollow.Content = reqData.ReplyMsgUnFollow
	config.SaveConfig()
	c.JSON(http.StatusOK, gin.H{
		"code": "ok",
	})
}

func Del(c *gin.Context) {

	for i := range config.UserInfo.ReplyMsgList {
		if id, _ := strconv.Atoi(c.Query("id")); config.UserInfo.ReplyMsgList[i].ID == id {
			config.UserInfo.ReplyMsgList = removeReplyMsg(config.UserInfo.ReplyMsgList, id)
			config.SaveConfig()
			c.JSON(http.StatusOK, gin.H{"code": "ok"})
			break
		}
	}

}

// removeReplyMsg 删除ReplyMsgList中指定ID的项
func removeReplyMsg(replyMsgList []ReplyMsg, idToRemove int) []ReplyMsg {
	var updatedList []ReplyMsg

	for _, msg := range replyMsgList {
		if msg.ID != idToRemove {
			updatedList = append(updatedList, msg)
		}
	}

	return updatedList
}

func Save(c *gin.Context) {

	var replyMsg struct {
		ID      int      `json:"ID"`
		Keys    []string `json:"keys"`    // 关键词列表
		Content string   `json:"content"` // 回复内容
	}

	// 使用ShouldBindJSON方法将JSON数据绑定到结构体
	if err := c.ShouldBindJSON(&replyMsg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 修改 而不是 增加
	for i, v := range config.UserInfo.ReplyMsgList {
		if replyMsg.ID == v.ID {
			config.UserInfo.ReplyMsgList[i].Keys = replyMsg.Keys
			config.UserInfo.ReplyMsgList[i].Content = replyMsg.Content
			config.SaveConfig()
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": "ok",
	})

}

func Add(c *gin.Context) {
	var reqData struct {
		ID       int    `json:"ID"`
		Keys     string `json:"keys"`
		ReplyMsg string `json:"replyMsg"`
	}

	//使用ShouldBindJSON方法将JSON数据绑定到结构体
	if err := c.ShouldBindJSON(&reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	replyMsg := ReplyMsg{
		ID:      int(time.Now().UnixMicro()),
		Keys:    strings.Split(reqData.Keys, "|"),
		Content: reqData.ReplyMsg,
	}

	config.UserInfo.ReplyMsgList = append(config.UserInfo.ReplyMsgList, replyMsg)
	config.SaveConfig()

	c.JSON(http.StatusOK, gin.H{
		"code": "ok",
	})
}

func Index(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{"userInfo": struct {
		MID              json.Number `json:"MID"`
		BName            string      `json:"BName"`
		ReplyMsgList     []ReplyMsg  `json:"ReplyMsgList"`     // 关注的粉丝回复内容列表
		ReplyMsgUnFollow string      `json:"ReplyMsgUnFollow"` // 没关注的粉丝回复内容
		MidFansNum       int         `json:"MidFansNum"`       // 粉丝总数
	}{
		MID:              config.UserInfo.MID,
		BName:            config.UserInfo.BName,
		ReplyMsgList:     config.UserInfo.ReplyMsgList,
		ReplyMsgUnFollow: config.UserInfo.ReplyMsgUnFollow.Content,
		MidFansNum:       len(config.UserInfo.MidFans),
	}})
}

func StartServer() {
	r := setupRouter()
	r.Run("0.0.0.0:9558")
}
