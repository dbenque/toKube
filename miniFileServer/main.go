package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type serverHandler struct {
}

var sHandler = serverHandler{}

func (s *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.FileServer(http.Dir("/")).ServeHTTP(w, r)
		return
	case "POST":
		fmt.Printf("POST: %v\n", *r)
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Printf("Error:%v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer file.Close()
		f, err := os.OpenFile("/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Printf("Error:%v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer f.Close()
		if _, err = io.Copy(f, file); err != nil {
			fmt.Printf("Error:%v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Printf("Upload completed: %s\n", handler.Filename)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	fmt.Println("Starting MiniFileServer")
	http.ListenAndServe(":80", &sHandler)
}
