package main

import (
	"html/template"
	"log"
	"msg-classifier/internal/handlers"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	SourcePath = "web/templates"
)

func init() {
	err := godotenv.Load("local.env")
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
	}
}

func main() {
	router := gin.Default()

	tmpl := template.Must(template.ParseGlob(SourcePath + "/**/*.html"))
	router.SetHTMLTemplate(tmpl)

	h := htmx.New()

	router.Use(func(c *gin.Context) {
		ctx := h.NewHandler(c.Writer, c.Request)
		c.Set("htmx", ctx)
		c.Next()
	})

	AddHandlers(router)

	_ = router.Run(":8080")
}

func AddHandlers(router *gin.Engine) {
	messageHandler := handlers.NewMsgHandler()
	webHandler := handlers.NewWebHandler(htmx.New())

	router.GET("/", webHandler.Home)

	api := router.Group("/api")
	api.POST("/message", messageHandler.ReceiveMessage)
}
