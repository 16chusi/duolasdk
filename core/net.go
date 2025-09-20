package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

// RequestInterceptor 请求拦截器函数类型
type RequestInterceptor func(req *http.Request) error

// ResponseInterceptor 响应拦截器函数类型
type ResponseInterceptor func(res *Response) error

// HttpCli 是我们的 axios 客户端实例
type HttpCli struct {
	log                  *AppLog
	httpClient           *http.Client
	baseURL              string
	headers              http.Header
	jar                  http.CookieJar
	requestInterceptors  []RequestInterceptor
	responseInterceptors []ResponseInterceptor
}

// Config 用于创建 HttpCli 实例的配置
type Config struct {
	BaseURL      string
	Headers      map[string]string
	ResponseType string
	Method       string
	HTTPClient   *http.Client
}

// Response 是对 http.Response 的封装
type Response struct {
	StatusCode int
	Header     http.Header
	Body       string
}

func NewHttp(log *AppLog) *HttpCli {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("无法创建 cookie jar: %s", err)
	}

	statefulClient := &http.Client{
		Jar: jar,
	}

	if log == nil {
		log = NewLogger(&LoggerOption{
			FileName: "log.log",
			Level:    "debug",
			Prefix:   "duola-sdk",
			Type:     "file",
		})
	}

	return &HttpCli{
		log:        log,
		httpClient: statefulClient,
		headers:    make(http.Header),
		jar:        jar,
	}
}

func (c *HttpCli) Create(cfg *Config) *HttpCli {
	c.log.Info("配置 HttpCli 实例", cfg)
	if cfg == nil {
		return c
	}

	if cfg.BaseURL != "" {
		// Ensure baseURL always ends with a slash for correct relative path resolution
		// This is crucial for url.JoinPath to work as expected when path starts with a slash.
		c.baseURL = strings.TrimSuffix(cfg.BaseURL, "/") + "/"
		c.log.Debug("HttpCli.Create: BaseURL after normalization", "normalizedURL", c.baseURL) // Added debug log
	}
	if cfg.Headers != nil {
		for k, v := range cfg.Headers {
			c.headers.Set(k, v)
		}
	}
	if cfg.HTTPClient != nil {
		c.httpClient = cfg.HTTPClient
	}

	return c
}

func (c *HttpCli) UseRequestInterceptor(interceptors ...RequestInterceptor) {
	c.requestInterceptors = append(c.requestInterceptors, interceptors...)
}

func (c *HttpCli) UseResponseInterceptor(interceptors ...ResponseInterceptor) {
	c.responseInterceptors = append(c.responseInterceptors, interceptors...)
}

type Options struct {
	Headers map[string]string
	Query   map[string]string
	Body    interface{}
}

func (c *HttpCli) Get(url string, options Options) (*Response, error) {
	return c.Do(http.MethodGet, url, options)
}

func (c *HttpCli) Post(url string, options Options) (*Response, error) {
	return c.Do(http.MethodPost, url, options)
}

func (c *HttpCli) Put(url string, options Options) (*Response, error) {
	return c.Do(http.MethodPut, url, options)
}

func (c *HttpCli) Delete(url string, options Options) (*Response, error) {
	return c.Do(http.MethodDelete, url, options)
}

func (c *HttpCli) Patch(url string, options Options) (*Response, error) {
	return c.Do(http.MethodPatch, url, options)
}

func (c *HttpCli) Head(url string, options Options) (*Response, error) {
	return c.Do(http.MethodHead, url, options)
}

func (c *HttpCli) Options(url string, options Options) (*Response, error) {
	return c.Do(http.MethodOptions, url, options)
}

func (c *HttpCli) Do(method, path string, opt Options) (*Response, error) {
	c.log.Debug("HttpCli.Do: Before JoinPath", "baseURL", c.baseURL, "path", path) // Added debug log
	// Use url.JoinPath for robust URL construction
	fullURL := ""
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		fullURL = path
	} else {
		p, err := url.JoinPath(c.baseURL, path)
		if err != nil {
			return nil, fmt.Errorf("无法拼接URL: %w", err)
		}
		fullURL = p
	}
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("错误的URL地址: %w", err)
	}

	if opt.Query != nil {
		q := parsedURL.Query()
		for k, v := range opt.Query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		parsedURL.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	var contentType string
	var requestBodyBytes []byte // To store body for logging

	if opt.Body != nil {
		headerContentType, isJSON := opt.Headers["Content-Type"]
		if isJSON && headerContentType == "application/json" {
			contentType = "application/json"
			jsonData, err := json.Marshal(opt.Body)
			if err != nil {
				return nil, fmt.Errorf("错误的json格式: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonData)
			requestBodyBytes = jsonData // Store for logging
		} else {
			contentType = "application/x-www-form-urlencoded"
			var bodyMap map[string]interface{}
			bodyBytes, _ := json.Marshal(opt.Body)
			_ = json.Unmarshal(bodyBytes, &bodyMap)
			formData := url.Values{}
			for key, val := range bodyMap {
				formData.Set(key, fmt.Sprintf("%v", val))
			}
			encodedData := formData.Encode()
			bodyReader = bytes.NewBufferString(encodedData)
			requestBodyBytes = []byte(encodedData) // Store for logging
		}
	}

	req, err := http.NewRequest(method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("无法创建请求: %w", err)
	}

	// Log the full request details
	c.log.Debug("HTTP Request", "Method", method, "URL", req.URL.String(), "Headers", req.Header, "Body", string(requestBodyBytes))

	for k, v := range c.headers {
		req.Header[k] = v
	}
	if opt.Headers != nil {
		for k, v := range opt.Headers {
			req.Header.Set(k, v)
		}
	}
	if opt.Body != nil {
		req.Header.Set("Content-Type", contentType)
	}

	for _, interceptor := range c.requestInterceptors {
		if err := interceptor(req); err != nil {
			return nil, fmt.Errorf("请求拦截器错误: %w", err)
		}
	}

	httpRes, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer httpRes.Body.Close()

	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	response := &Response{
		StatusCode: httpRes.StatusCode,
		Header:     httpRes.Header,
		Body:       string(resBody),
	}

	for _, interceptor := range c.responseInterceptors {
		if err := interceptor(response); err != nil {
			return nil, fmt.Errorf("响应拦截器错误: %w", err)
		}
	}

	return response, nil
}

// PostStream 发送 POST 请求并返回原始的 http.Response 用于流式处理
func (c *HttpCli) PostStream(path string, options Options) (*http.Response, error) {
	c.log.Debug("HttpCli.PostStream: Before JoinPath", "baseURL", c.baseURL, "path", path) // Added debug log
	// Use url.JoinPath for robust URL construction
	fullURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("无法拼接URL: %w", err)
	}

	var bodyReader io.Reader
	var requestBodyBytes []byte // To store body for logging

	if options.Body != nil {
		jsonData, err := json.Marshal(options.Body)
		if err != nil {
			return nil, fmt.Errorf("错误的json格式: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
		requestBodyBytes = jsonData // Store for logging
	}

	req, err := http.NewRequest(http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("无法创建请求: %w", err)
	}

	// Log the full request details
	c.log.Debug("HTTP Stream Request", "Method", http.MethodPost, "URL", req.URL.String(), "Headers", req.Header, "Body", string(requestBodyBytes))

	for k, v := range c.headers {
		req.Header[k] = v
	}
	if options.Headers != nil {
		for k, v := range options.Headers {
			req.Header.Set(k, v)
		}
	}
	if options.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, interceptor := range c.requestInterceptors {
		if err := interceptor(req); err != nil {
			return nil, fmt.Errorf("请求拦截器错误: %w", err)
		}
	}

	return c.httpClient.Do(req)
}

func (c *HttpCli) GetCaptchaImage(path string) (string, error) {
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("错误的URL地址: %w", err)
	}
	fullURL, err = fullURL.Parse(path)
	if err != nil {
		return "", fmt.Errorf("错误的请求地址: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, fullURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("无法创建验证码请求: %w", err)
	}

	httpRes, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取验证码图片失败: %w", err)
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取验证码失败，状态码: %d", httpRes.StatusCode)
	}

	imageBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return "", fmt.Errorf("读取验证码图片数据失败: %w", err)
	}

	encodedString := base64.StdEncoding.EncodeToString(imageBytes)

	mimeType := http.DetectContentType(imageBytes)
	if mimeType == "application/octet-stream" {
		mimeType = "image/png"
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encodedString)
	return dataURL, nil
}

// DownloadFileWithResume 支持断点续传的文件下载方法
func (c *HttpCli) DownloadFileWithResume(url string, filepath string, progressCallback func(int64, int64)) error {
	tmpFilepath := filepath + ".tmp"

	// 检查临时文件是否存在
	fileInfo, err := os.Stat(tmpFilepath)
	var file *os.File
	var startOffset int64 = 0
	if err == nil {
		// 文件存在，断点续传
		startOffset = fileInfo.Size()
		file, err = os.OpenFile(tmpFilepath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("无法打开文件进行追加: %w", err)
		}
	} else {
		// 文件不存在，创建新文件
		file, err = os.Create(tmpFilepath)
		if err != nil {
			return fmt.Errorf("无法创建文件: %w", err)
		}
	}
	defer file.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("无法创建请求: %w", err)
	}

	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法发送请求: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("错误的响应状态: %s", resp.Status)
	}

	totalSize := resp.ContentLength + startOffset

	// 使用缓冲区控制内存使用
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	written := startOffset

	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := file.Write(buf[0:nr])
			if ew != nil {
				return fmt.Errorf("无法写入文件: %w", ew)
			}
			if nr != nw {
				return fmt.Errorf("写入文件字节数不足")
			}
			written += int64(nw)
			if progressCallback != nil {
				progressCallback(written, totalSize)
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return fmt.Errorf("无法读取响应体: %w", er)
		}
	}

	// 将临时文件重命名为最终文件
	err = os.Rename(tmpFilepath, filepath)
	if err != nil {
		return fmt.Errorf("无法重命名临时文件: %w", err)
	}

	return nil
}
