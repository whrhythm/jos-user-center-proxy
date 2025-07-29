package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	// 初始化数据库
	var err error
	DB, err = InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 测试连接
	if err := testConnection(); err != nil {
		log.Fatalf("Database connection test failed: %v", err)
	}
}

func testConnection() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func main() {
	targetBaseURL := "http://join-user-center:8084" // 目标服务地址

	// 创建带连接池的HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	http.HandleFunc("/", proxyHandler(targetBaseURL, client))

	// 启动服务器
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	log.Println("Starting proxy server on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server error:", err)
	}
}

// proxyHandler returns an http.HandlerFunc that proxies requests to the targetBaseURL using the provided client.
func proxyHandler(targetBaseURL string, client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s", r.Method, r.URL.Path)

		targetPath := rewritePath(r.URL.Path)
		targetURL, err := buildTargetURL(targetBaseURL, targetPath, r.URL.RawQuery)
		if err != nil {
			http.Error(w, "Invalid target path", http.StatusBadRequest)
			return
		}

		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body)) // 重置请求体

		// 如果targetPath == 'organization/user', 并且method是POST
		if targetPath == "/organization/user" && r.Method == http.MethodPost {
			log.Println("Handling POST request to /organization/user")
			// 这里可以添加特定的处理逻辑
			if err := syncUser(r, body); err != nil {
				log.Printf("Error syncing user: %v", err)
				http.Error(w, "Failed to sync user", http.StatusInternalServerError)
				return
			}
		}

		req, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		copyHeaders(req, r)
		copyCookies(req, r)

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Backend request failed: %v", err)
			http.Error(w, "Service unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		log.Printf("Forwarded to %s in %v", targetURL, time.Since(start))

		copyResponse(w, resp)
	}
}

// rewritePath removes the "/prod" prefix if present.
func rewritePath(path string) string {
	if after, ok := strings.CutPrefix(path, "/prod"); ok {
		// newPath := "/" + after
		newPath := after
		log.Printf("Rewriting path: %s -> %s", path, newPath)
		return newPath
	}
	return path
}

// buildTargetURL constructs the target URL with query parameters.
func buildTargetURL(base, path, rawQuery string) (string, error) {
	targetURL, err := url.JoinPath(base, path)
	if err != nil {
		return "", err
	}
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}
	return targetURL, nil
}

// copyHeaders copies all headers from the original request to the new request.
func copyHeaders(dst *http.Request, src *http.Request) {
	for key, values := range src.Header {
		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}
}

// copyCookies copies all cookies from the original request to the new request.
func copyCookies(dst *http.Request, src *http.Request) {
	for _, cookie := range src.Cookies() {
		dst.AddCookie(cookie)
	}
}

// copyResponse copies the response headers, status code, and body to the ResponseWriter.
func copyResponse(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Stream error: %v", err)
	}
}
