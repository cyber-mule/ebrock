package main

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"log"
	"net/http"
)

var (
	//go:embed download.html
	DownloadHtml []byte
)

type ErrorResponse struct {
	Msg string
}

func (e *ErrorResponse) Error() string {
	if len(e.Msg) == 0 {
		return `{"error": "unknown error"}`
	}

	return `{"error": "` + e.Msg + `"}`
}

func initServer() {
	log.Println("Initializing server...")
	http.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write(DownloadHtml)
	})
	http.HandleFunc("/submit", submitURL)
	http.HandleFunc("/v1/direct", directDownload)

	if err := http.ListenAndServe(":8880", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// submitURL 提交url
func submitURL(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		// 返回错误信息, 通过结构体进行响应
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "非法请求"}
		_, _ = w.Write([]byte(response.Error()))
		log.Println("Received invalid request: empty URL")
		return
	}

	// 检查url是否合法是一个可以访问的url
	// 这里可以使用正则表达式进行检查
	response, err := http.Get(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "给定地址无法访问"}
		_, _ = w.Write([]byte(response.Error()))
		log.Printf("URL is not accessible: %v", err)
		return
	}
	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	// 检查响应大小是否超过2m
	if response.ContentLength > 2*1024*1024 {
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "响应内容超过2M"}
		_, _ = w.Write([]byte(response.Error()))
		log.Println("Response content exceeds 2MB")
		return
	}

	s := base64.StdEncoding.EncodeToString([]byte(url))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"data": "/v1/direct?context=` + s + `"}`))
	log.Println("URL submission processed successfully")
}

func directDownload(w http.ResponseWriter, r *http.Request) {
	s := r.FormValue("context")
	if s == "" {
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "非法请求"}
		_, _ = w.Write([]byte(response.Error()))
		log.Println("Received invalid request: empty context")
		return
	}

	url, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "非法请求"}
		_, _ = w.Write([]byte(response.Error()))
		log.Printf("Failed to decode context: %v", err)
		return
	}

	log.Printf("Direct download for URL: %s", url)
	response, err := http.Get(string(url))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := &ErrorResponse{Msg: "给定地址无法访问"}
		_, _ = w.Write([]byte(response.Error()))
		log.Printf("URL is not accessible: %v", err)
		return
	}
	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(response.StatusCode)
	_, err = bufio.NewReader(response.Body).WriteTo(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := &ErrorResponse{Msg: "下载失败"}
		_, _ = w.Write([]byte(response.Error()))
		log.Printf("Failed to write response: %v", err)
		return
	}

	log.Println("Direct download processed successfully")
}

func main() {
	initServer()
}
