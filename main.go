package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
)

type Data struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
type Message struct {
	Content string `json:"content"`
}
type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}
type result struct {
	Choices []Choice `json:"choices"`
}

func main() {
	r := gin.Default()
	r.Use(corsMiddleware())

	r.POST("/session", func(c *gin.Context) {

		fmt.Printf("请求方法：%s\n", c.Request.Method)
		fmt.Printf("请求路径：%s\n", c.Request.URL.Path)
		fmt.Printf("请求参数：%v\n", c.Request.Form)

		data := Data{
			Status:  "Success",
			Message: "",
			Data: map[string]interface{}{
				"auth":  false,
				"model": "ChatGPTAPI",
			}}

		c.JSON(200, data)
	})
	r.POST("/chat-process", func(c *gin.Context) {
		all, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return
		}
		fmt.Printf(string(all))
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": string(all)},
			},
			"temperature": 0.7,
		}
		marshal, err := json.Marshal(requestData)
		if err != nil {
			return
		}
		// 使用系统代理构建Transport
		proxyURL, err := url.Parse("socks5://127.0.0.1:7890") // 替换为实际的代理URL
		if err != nil {
			c.String(http.StatusBadRequest, "无效的代理URL")
			return
		}

		// 创建带有代理的Transport
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		// 创建带有自定义Transport的Client
		client := &http.Client{
			Transport: transport,
		}

		// 构建POST请求
		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(marshal))
		if err != nil {
			c.String(http.StatusInternalServerError, "无法创建请求")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer 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")
		req.Header.Set("proxy_buffering", "off")

		// 发送POST请求
		resp, err := client.Do(req)
		if err != nil {
			c.String(http.StatusInternalServerError, "请求发送失败")
			return
		}
		defer resp.Body.Close()

		// 处理响应
		var resultData result
		err = json.NewDecoder(resp.Body).Decode(&resultData)
		if err != nil {
			c.String(http.StatusInternalServerError, "无法解析响应数据")
			return
		}
		fmt.Println()
		fmt.Println(resultData.Choices[0].Message.Content)
		c.JSON(http.StatusOK, gin.H{
			"message": resultData.Choices[0].Message.Content,
		})
	})
	r.Run(":8888")
}

// 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	}
}
