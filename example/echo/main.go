package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/3JoB/httpfileserver"
)

func main() {
	// Any request to /static/somefile.txt will serve the file at the location ./somefile.txt
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("*", httpfileserver.New("/assets/", ".").EchoHandle())
	e.Logger.Fatal(e.Start(":1113"))
}
