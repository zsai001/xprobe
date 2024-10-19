package util

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"strings"
)

func IsHTTPS(c *gin.Context) bool {
	// 检查 X-Forwarded-Proto
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return true
	}

	// 检查 Cloudflare 特有的头
	if c.GetHeader("Cf-Visitor") != "" {
		var cfVisitor map[string]string
		json.Unmarshal([]byte(c.GetHeader("Cf-Visitor")), &cfVisitor)
		if scheme, ok := cfVisitor["scheme"]; ok && scheme == "https" {
			return true
		}
	}

	// 检查 Cloudflare 的另一个头
	if c.GetHeader("Cf-Request-Scheme") == "https" {
		return true
	}

	return false
}

func DebugRequest(c *gin.Context) {
	var debugInfo strings.Builder

	// Request Method and URL
	debugInfo.WriteString(fmt.Sprintf("Request Method: %s\n", c.Request.Method))
	debugInfo.WriteString(fmt.Sprintf("Request URL: %s\n", c.Request.URL))

	// Headers
	debugInfo.WriteString("Headers:\n")
	for key, values := range c.Request.Header {
		for _, value := range values {
			debugInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// GET Query Parameters
	debugInfo.WriteString("GET Query Parameters:\n")
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			debugInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// POST Form Data
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		if err := c.Request.ParseForm(); err == nil {
			debugInfo.WriteString("POST/PUT/PATCH Form Data:\n")
			for key, values := range c.Request.PostForm {
				for _, value := range values {
					debugInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
				}
			}
		}
	}

	// Request Body
	if c.Request.Body != nil {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err == nil {
			c.Request.Body = ioutil.NopCloser(strings.NewReader(string(bodyBytes))) // Reset the body
			debugInfo.WriteString("Request Body:\n")
			debugInfo.WriteString(fmt.Sprintf("  %s\n", string(bodyBytes)))

			// Try to parse JSON
			var jsonData interface{}
			if json.Unmarshal(bodyBytes, &jsonData) == nil {
				prettyJSON, err := json.MarshalIndent(jsonData, "  ", "    ")
				if err == nil {
					debugInfo.WriteString("Parsed JSON Body:\n")
					debugInfo.WriteString(fmt.Sprintf("  %s\n", string(prettyJSON)))
				}
			}
		}
	}

	// Log the debug info
	fmt.Println(debugInfo.String())

	// Continue with the request
	c.Next()
}
