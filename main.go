package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Data struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
type Message struct {
	Content string `json:"content"`
}

func main() {
	_ = godotenv.Load()
	r := gin.Default()
	r.Use(corsMiddleware())

	r.POST("/session", func(c *gin.Context) {
		data := Data{
			Status:  "Success",
			Message: "",
			Data: map[string]interface{}{
				"auth":  false,
				"model": "ChatGPTAPI",
			}}

		c.JSON(200, data)
	})
	r.POST("/chat-process", process)

	r.Run(":8888")
}

func process(c *gin.Context) {
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return
	}
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": string(all)},
		},
		"temperature": 0.7,
		"stream":      true,
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
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	//req.Header.Set("proxy_buffering", "off")
	// 发送POST请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusInternalServerError, "请求发送失败")
		return
	}
	defer resp.Body.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Connection", "keep-alive")
	c.Header("Cache-Control", "no-cache")
	scanner := bufio.NewScanner(resp.Body)
	str := ""
	c.Stream(func(w io.Writer) bool {

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				responseBuffer := []byte(line[6:])
				var responseChunk map[string]interface{}
				err := json.Unmarshal(responseBuffer, &responseChunk)
				if err != nil {
					fmt.Println("解码响应块时出错:", err)
					return false
				}
				choices, ok := responseChunk["choices"].([]interface{})
				if !ok || len(choices) == 0 {
					return false
				}
				choice, ok := choices[0].(map[string]interface{})
				if !ok {
					return false
				}
				delta, ok := choice["delta"].(map[string]interface{})
				if !ok {
					return false
				}
				content, ok := delta["content"].(string)
				if !ok {
					return false

				}
				str += content
				responseChunk["text"] = str
				jsonData, err := json.Marshal(responseChunk)
				if err != nil {
					fmt.Println("转换为 JSON 失败:", err)
					return false
				}
				fmt.Fprintf(c.Writer, "%s\n", jsonData)
				c.Writer.Flush()
			}
		}
		return true
	})

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
