package main

import (
	"github.com/gin-gonic/gin"

	"github.com/3JoB/httpfileserver"
)

func main() {
	r := gin.New()
	r.GET("*", httpfileserver.New("/assets/", ".").GinHandle())
	r.Run(":1122")
}
