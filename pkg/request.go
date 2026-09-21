package pkg

import (
	"github.com/gin-gonic/gin"
)

type Request struct {
	Method  string
	Url     string
	Body    string
	Headers map[string]string
}

func ReturnJson(c *gin.Context, status int, body any) {
	if status >= 200 && status < 300 {
		c.JSON(status, gin.H{
			"data": body,
		})
		return
	}

	c.JSON(status, gin.H{
		"error": body,
	})
}
