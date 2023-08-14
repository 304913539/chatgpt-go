package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
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
		req.Header.Set("Authorization", "Bearer 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")
		req.Header.Set("proxy_buffering", "off")

		// 发送POST请求
		resp, err := client.Do(req)
		if err != nil {
			c.String(http.StatusInternalServerError, "请求发送失败")
			return
		}
		defer resp.Body.Close()
		//fmt.Println(resp.Body)
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
		//c.Stream(func(w io.Writer) bool {
		//	// 将响应写入到前端
		//	_, err := w.Write(responseBody)
		//	if err != nil {
		//		fmt.Println(err.Error())
		//		// 处理错误情况
		//		return false
		//	}
		//	return true
		//})
		//c.Header("Content-Type", "text/event-stream")
		//c.Header("Cache-Control", "no-cache")
		//c.Header("Connection", "keep-alive")
		//c.Header("Access-Control-Allow-Origin", "*")
		//responseBody, _ := ioutil.ReadAll(resp.Body)
		//
		//c.Stream(func(w io.Writer) bool {
		//	// 将响应写入到前端
		//	_, err := w.Write(responseBody)
		//	if err != nil {
		//		fmt.Println(err.Error())
		//		// 处理错误情况
		//		return false
		//	}
		//	return true
		//})
	})
	//r.POST("/chat-process", func(c *gin.Context) {
	//	// 设置响应头为 text/event-stream
	//	c.Header("Content-Type", "text/event-stream")
	//	c.Header("Cache-Control", "no-cache")
	//	c.Header("Connection", "keep-alive")
	//	c.Header("Access-Control-Allow-Origin", "*")
	//
	//	c.Stream(func(w io.Writer) bool {
	//		// 将响应写入到前端
	//		_, err := w.Write(responseBody)
	//		if err != nil {
	//			// 处理错误情况
	//			return false
	//		}
	//		return true
	//	})
	//})
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
func chatProcessHandler(c *gin.Context) {
	c.Header("Content-Type", "application/octet-stream")

	var requestData struct {
		Prompt        string         `json:"prompt"`
		Options       RequestOptions `json:"options"`
		SystemMessage string         `json:"systemMessage"`
		Temperature   float64        `json:"temperature"`
		TopP          float64        `json:top_p`
	}

	err := c.BindJSON(&requestData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	firstChunk := true

	err = chatReplyProcess(requestData.Prompt, requestData.Options, requestData.SystemMessage, requestData.Temperature, requestData.TopP, func(chat ChatMessage) {
		if firstChunk {
			c.JSON(http.StatusOK, chat)
			firstChunk = false
		} else {
			c.Writer.WriteString("\n")
			c.JSON(http.StatusOK, chat)
		}
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

type RequestOptions struct {
	Message         string            `json:"message"`
	LastContext     interface{}       `json:"lastContext"`
	Process         func(interface{}) `json:"-"`
	SystemMessage   string            `json:"systemMessage"`
	Temperature     float64           `json:"temperature"`
	TopP            float64           `json:"top_p"`
	ParentMessageId string            `json:"parentMessageId"`
}
type SendMessageOptions struct {
	TimeoutMs        int    `json:"timeoutMs"`
	SystemMessage    string `json:"systemMessage,omitempty"`
	CompletionParams struct {
		Model       string  `json:"model,omitempty"`
		Temperature float64 `json:"temperature,omitempty"`
		TopP        float64 `json:"top_p,omitempty"`
	} `json:"completionParams,omitempty"`
	ParentMessageId string `json:"parentMessageId,omitempty"`
	Stream          bool   `json:"-"`
}
type CompletionParams struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}
type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
	System    Role = "system"
)

type ChatMessage struct {
	ID              string      `json:"id"`
	Text            string      `json:"text"`
	Role            Role        `json:"role"`
	Name            string      `json:"name,omitempty"`
	Delta           string      `json:"delta,omitempty"`
	Detail          interface{} `json:"detail,omitempty"`
	ParentMessageID string      `json:"parentMessageId,omitempty"`
	ConversationID  string      `json:"conversationId,omitempty"`
}

func chatReplyProcess(message string, lastContext RequestOptions, systemMessage string, temperature float64, topP float64, process func(ChatMessage)) error {
	options := SendMessageOptions{TimeoutMs: 100000}
	var apiModel = "ChatGPTAPI" // Set the value of apiModel here

	if apiModel == "ChatGPTAPI" {
		if systemMessage != "" {
			options.SystemMessage = systemMessage
		}
		options.CompletionParams = struct {
			Model       string  `json:"model,omitempty"`
			Temperature float64 `json:"temperature,omitempty"`
			TopP        float64 `json:"top_p,omitempty"`
		}(CompletionParams{Model: apiModel, Temperature: temperature, TopP: topP})
	}

	if !reflect.DeepEqual(lastContext, RequestOptions{}) {
		if apiModel == "ChatGPTAPI" {
			options.ParentMessageId = lastContext.ParentMessageId
		} else {
			options.ParentMessageId = lastContext.ParentMessageId
		}
	}

	response, err := api.SendMessage(message, options)
	if err != nil {
		if codeMsg, ok := ErrorCodeMessage[err.StatusCode]; ok {
			return fmt.Errorf(codeMsg)
		}
		return fmt.Errorf(err.Message)
	}

	process(response)

	return nil
}

func SendMessage(text string, opts sendMessageOptions) (response, error) {
	messageID := opts.MessageID
	if messageID == "" {
		messageID = uuidv4()
	}
	timeoutMs := opts.TimeoutMs
	onProgress := opts.OnProgress
	stream := opts.Stream && onProgress != nil
	completionParams := opts.CompletionParams

	var abortSignal chan bool
	var abortController chan bool
	if timeoutMs > 0 && abortSignal == nil {
		abortController = make(chan bool)
		abortSignal = abortController
	}

	message := message{
		Role:            "user",
		ID:              messageID,
		ParentMessageID: opts.ParentMessageID,
		Text:            text,
	}
	err := upsertMessage(message)
	if err != nil {
		return response{}, err
	}

	messages, maxTokens, numTokens, err := buildMessages(text, opts)
	if err != nil {
		return response{}, err
	}

	result := response{
		Role:            "assistant",
		ID:              uuidv4(),
		ParentMessageID: messageID,
		Text:            "",
	}

	responseP := make(chan response)
	go func() {
		url := fmt.Sprintf("%s/chat/completions", apiBaseUrl)
		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		body := map[string]interface{}{
			"max_tokens": maxTokens,
			"messages":   messages,
			"stream":     stream,
		}
		if completionParams != nil {
			for k, v := range completionParams {
				body[k] = v
			}
		}

		if debug {
			fmt.Printf("sendMessage (%d tokens): %v\n", numTokens, body)
		}

		if stream {
			onMessage := func(data string) {
				if data == "[DONE]" {
					result.Text = result.Text
					responseP <- result
					return
				}

				var responseData map[string]interface{}
				err := json.Unmarshal([]byte(data), &responseData)
				if err != nil {
					fmt.Println("OpenAI stream SEE event unexpected error", err)
					return
				}

				responseID, ok := responseData["id"].(string)
				if ok {
					result.ID = responseID
				}

				choices, ok := responseData["choices"].([]interface{})
				if ok && len(choices) > 0 {
					delta, ok := choices[0].(map[string]interface{})["delta"].(map[string]interface{})
					if ok {
						resultDelta, ok := delta["content"].(string)
						if ok {
							result.Text += resultDelta
						}
					}
				}

				result.Detail = responseData

				if deltaRole, ok := delta["role"].(string); ok {
					result.Role = deltaRole
				}

				onProgress(result)
			}

			err := fetchSSE(url, headers, body, abortSignal, onMessage)
			if err != nil {
				responseP <- response{}
			}
		} else {
			res, err := fetch(url, "POST", headers, body, abortSignal)
			if err != nil {
				return response{}, err
			}

			if !res.Ok {
				reason, _ := ioutil.ReadAll(res.Body)
				msg := fmt.Sprintf("OpenAI error %d: %s", res.Status, string(reason))
				err = fmt.Errorf(msg)
				return response{}, err
			}

			var responseData map[string]interface{}
			err = json.NewDecoder(res.Body).Decode(&responseData)
			if err != nil {
				return response{}, err
			}

			responseID, ok := responseData["id"].(string)
			if ok {
				result.ID = responseID
			}

			choices, ok := responseData["choices"].([]interface{})
			if ok && len(choices) > 0 {
				message2, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
				if ok {
					result.Text, _ = message2["content"].(string)
				}
			} else {
				res2 := responseData
				err = fmt.Errorf("OpenAI error: %v", res2["detail"])
				return response{}, err
			}

			result.Detail = responseData
			responseP <- result
		}
	}()

	responseTimeout := make(chan bool)
	go func() {
		timeoutMs := timeoutMs
		if timeoutMs > 0 {
			milliseconds := time.Duration(timeoutMs) * time.Millisecond
			time.Sleep(milliseconds)
			responseTimeout <- true
		}
	}()

	select {
	case <-responseP:
		if abortController != nil {
			close(abortController)
		}
		return response{}, fmt.Errorf("OpenAI timed out waiting for response")
	case <-responseTimeout:
		if abortController != nil {
			close(abortController)
		}
		return response, nil
	}
}
