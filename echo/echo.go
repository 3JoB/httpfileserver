package echo

import (
	"github.com/3JoB/httpfileserver"
	"github.com/labstack/echo/v4"
)

func EchoHandle(fs *httpfileserver.FileServer) echo.HandlerFunc{
	return func(c echo.Context) error {
		fs.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}