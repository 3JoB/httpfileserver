package main

import (
	"net/http"

	"github.com/3JoB/httpfileserver"
)

func main() {
	http.Handle("/new/", httpfileserver.New("/new/", "."))
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":1113", nil)
}
