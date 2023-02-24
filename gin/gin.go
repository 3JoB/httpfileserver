package gin

import (
	"github.com/3JoB/httpfileserver"
	"github.com/gin-gonic/gin"
)

func GinHandle(fs *httpfileserver.FileServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		fs.ServeHTTP(c.Writer, c.Request)
	}
}