package main

import (
	"net/http"

	"github.com/3JoB/httpfileserver"
)

func main() {
	var fs httpfileserver.FileServer
	fs.Config.FlateLevel = 3
	http.Handle("/new/", httpfileserver.New("/new/", "."))
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":1113", nil)
}