package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"io"
	"log"
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
type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}
type result struct {
	Choices []Choice `json:"choices"`
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
		c.String(http.StatusInternalServerError, "请求发送失败")
		return
	}
	defer resp.Body.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Connection", "keep-alive")
	c.Header("Cache-Control", "no-cache")

	c.Stream(func(w io.Writer) bool {

		// Simulate handlerFunction's response for demonstration
		//readAll, err := io.ReadAll(resp.Body)
		buf := make([]byte, 1024)

		//if err != nil {
		//	return false
		//}
		//lines := strings.Split(string(readAll), "\n\n")
		str := ""
		var buffer bytes.Buffer // 用于存储未处理完的数据块

		for {
			n, _ := resp.Body.Read(buf)
			if n <= 0 {
				return false
			}
			buffer.Write(buf[:n]) // 将新数据块写入缓冲区
			for {
				line, err := buffer.ReadString('\n') // 尝试从缓冲区中读取一行数据
				if err != nil {
					break // 无法读取完整的一行，继续等待更多数据
				}

				// 在这里处理完整的数据行，可以添加你的逻辑
				message := strings.Replace(line, "data: ", "", 1)

				if message == "[DONE]" {
					return false
				}

				var parsed struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				fmt.Println(message)
				if err := json.Unmarshal([]byte(message), &parsed); err != nil {
					fmt.Println("Error parsing JSON:", err)
					continue
				} else {
					fmt.Println("11112ss", parsed.Choices[0].Delta.Content)
				}

				// 将 JSON 字符串解析为 map[string]interface{} 类型
				var jsonData map[string]interface{}

				err = json.Unmarshal([]byte(message), &jsonData)
				if err != nil {
					log.Fatal(err)
				}

				str = str + parsed.Choices[0].Delta.Content
				// 添加新的键值对到 JSON 对象中
				jsonData["text"] = str

				// 将更新后的 JSON 对象转换回 JSON 字符串
				updatedJSON, err := json.Marshal(jsonData)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Fprintf(c.Writer, "%s\n", updatedJSON)

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
