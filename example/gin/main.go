package main

import (
	"github.com/gin-gonic/gin"

	"github.com/3JoB/httpfileserver"
	hg "github.com/3JoB/httpfileserver/gin"
)

func main() {
	r := gin.New()
	r.GET("*", hg.GinHandle(httpfileserver.New("/assets/", ".")))
	r.Run(":1122")
}
