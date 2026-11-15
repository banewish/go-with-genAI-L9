package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		var bodyStr string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				var body map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &body); err == nil {
					delete(body, "password")
					redactedBody, _ := json.Marshal(body)
					bodyStr = string(redactedBody)
				} else {
					bodyStr = string(bodyBytes)
				}
			}
		}

		c.Next()

		statusCode := c.Writer.Status()
		duration := time.Since(startTime)

		logStr := "[" + method + "] " + path + " " + query
		if bodyStr != "" {
			logStr += " (body: " + bodyStr + ")"
		}
		logStr += " -> " + strconv.Itoa(statusCode) + " (" + strconv.FormatInt(duration.Milliseconds(), 10) + "ms)"

		log.Println(logStr)
	}
}
