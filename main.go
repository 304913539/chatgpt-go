package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
		req.Header.Set("proxy_buffering", "off")
		// 发送POST请求
		resp, err := client.Do(req)
		if err != nil {
			c.String(http.StatusInternalServerError, "请求发送失败")
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 1024)
		_, _ = c.Writer.Write([]byte("["))

		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				handler(c, buf[:n])
			} else {
				break
			}
			if err != nil {
				break
			}
			time.Sleep(time.Second)
		}
		_, _ = c.Writer.Write([]byte("]"))

		//for {
		//
		//	readAll, err := io.ReadAll(resp.Body)
		//	if err != nil {
		//		log.Fatalln(err.Error())
		//		return
		//	}
		//	fmt.Println(string(readAll))
		//
		//	if resp.StatusCode != http.StatusOK {
		//		fmt.Println(string(readAll))
		//		return
		//	}
		//	handler(c, readAll)
		//
		//}
	})

	r.Run(":8888")
}
func handler(c *gin.Context, respData []byte) {
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Connection", "keep-alive")
	c.Header("Cache-Control", "no-cache")

	//c.Status(http.StatusOK)
	c.Stream(func(w io.Writer) bool {

		// Simulate handlerFunction's response for demonstration
		lines := strings.Split(string(respData), "\n\n")
		str := ""
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

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

			if err := json.Unmarshal([]byte(message), &parsed); err != nil {
				fmt.Println("Error parsing JSON:", err)
				continue
			}

			fmt.Println("parsed content -", message)

			// 将 JSON 字符串解析为 map[string]interface{} 类型
			var jsonData map[string]interface{}
			err := json.Unmarshal([]byte(message), &jsonData)
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
			time.Sleep(time.Millisecond * 10) // Simulate typing effect delay

		}
		return true
	})
}

func qq(c *gin.Context) {
	jsonData := `
		[{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"","delta":"","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你","delta":"你","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"你"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好","delta":"好","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"好"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！","delta":"！","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"！"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有","delta":"有","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"有"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什","delta":"什","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"什"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么","delta":"么","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"么"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我","delta":"我","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"我"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以","delta":"可以","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"可以"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮","delta":"帮","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"帮"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助","delta":"助","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"助"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助你","delta":"你","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"你"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助你的","delta":"的","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"的"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助你的吗","delta":"吗","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"吗"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助你的吗？","delta":"？","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"？"},"finish_reason":null}]}},
{"role":"assistant","id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","parentMessageId":"090b183b-d6e1-47eb-81fc-7d72e1571732","text":"你好！有什么我可以帮助你的吗？","detail":{"id":"chatcmpl-7oC1WqDpjdJxOAU1z5HP6rrUfFlHM","object":"chat.completion.chunk","created":1692196974,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}}]
	`
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		c.String(500, "Failed to parse JSON")
		return
	}

	for _, obj := range data {
		updatedJSON, err := json.Marshal(obj)
		if err != nil {
			c.String(500, "Failed to marshal JSON")
			return
		}

		c.String(200, "%s\n", updatedJSON)
		c.Writer.Flush()
	}
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
