package duolasdk

//网络能力支持
import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	ResponseType string // 可选，指定响应类型，如 "json", "text" 等
	Method       string // 可选，默认是 GET，Post, Put, Delete 等
	HTTPClient   *http.Client
}

// Response 是对 http.Response 的封装
type Response struct {
	StatusCode int
	Header     http.Header
	Body       string
	//Request    *http.Request
}

func NewHttp(log *AppLog) *HttpCli {
	// ✅ 3. 核心修改在这里！
	// 创建一个新的 cookie jar。`nil` 表示使用默认选项。
	jar, err := cookiejar.New(nil)
	if err != nil {
		// 在实际应用中，这里几乎不可能出错，但好的习惯是处理它
		log.Fatalf("无法创建 cookie jar: %s", err)
	}

	// 创建一个新的 http.Client，并将我们的 jar 关联上
	// 不要再使用 http.DefaultClient，因为它是一个全局共享的客户端
	statefulClient := &http.Client{
		Jar: jar,
	}
	// 返回一个持有这个“有记忆”的客户端的 HttpCli 实例
	return &HttpCli{
		log:        log,
		httpClient: statefulClient, // 使用我们新建的客户端
		headers:    make(http.Header),
		jar:        jar, // 保存 jar 的引用
	}
}

// Create 创建一个新的 axios 客户端实例，类似 axios.create()
func (c *HttpCli) Create(cfg *Config) *HttpCli {
	c.log.Info("配置 HttpCli 实例", cfg)
	if cfg == nil {
		return c
	}

	if cfg.BaseURL != "" {
		c.baseURL = cfg.BaseURL
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

// UseRequestInterceptor 添加一个或多个请求拦截器
func (c *HttpCli) UseRequestInterceptor(interceptors ...RequestInterceptor) {
	c.requestInterceptors = append(c.requestInterceptors, interceptors...)
}

// UseResponseInterceptor 添加一个或多个响应拦截器
func (c *HttpCli) UseResponseInterceptor(interceptors ...ResponseInterceptor) {
	c.responseInterceptors = append(c.responseInterceptors, interceptors...)
}

// Options 封装单次请求的选项
type Options struct {
	Headers map[string]string
	Query   map[string]string
	Body    interface{} // 可以是任意可以被 json.Marshal 的类型
}

// Get 发送 GET 请求
func (c *HttpCli) Get(url string, options Options) (*Response, error) {
	return c.Do(http.MethodGet, url, options)
}

// Post 发送 POST 请求
func (c *HttpCli) Post(url string, options Options) (*Response, error) {
	return c.Do(http.MethodPost, url, options)
}

// Put 发送 PUT 请求
func (c *HttpCli) Put(url string, options Options) (*Response, error) {
	return c.Do(http.MethodPut, url, options)
}

// Delete 发送 DELETE 请求
func (c *HttpCli) Delete(url string, options Options) (*Response, error) {
	return c.Do(http.MethodDelete, url, options)
}

// Do 是实际执行请求的核心方法
func (c *HttpCli) Do(method, path string, opt Options) (*Response, error) {

	// --- 1. 构建完整的 URL ---
	// 这一步先做，确保后续日志和请求能用到完整的 URL
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("错误的URL地址: %w", err)
	}
	fullURL, err = fullURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("错误的请求地址: %w", err)
	}

	// 添加 Query 参数
	if opt.Query != nil {
		q := fullURL.Query()
		// Go 的 map 遍历是无序的，如果后端对参数顺序有要求，需要注意
		for k, v := range opt.Query {
			// 将 Query 的值也安全地转为字符串
			q.Set(k, fmt.Sprintf("%v", v))
		}
		fullURL.RawQuery = q.Encode()
	}

	// --- 2. 准备请求体 (Body) 和内容类型 (Content-Type) ---
	var bodyReader io.Reader
	var contentType string

	if opt.Body != nil {
		// 检查请求头是否强制指定了 JSON
		headerContentType, isJSON := opt.Headers["Content-Type"]
		if isJSON && headerContentType == "application/json" {
			// --- 发送 JSON 数据 ---
			contentType = "application/json"
			jsonData, err := json.Marshal(opt.Body)
			if err != nil {
				return nil, fmt.Errorf("错误的json格式: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonData)
			c.log.Infof("准备请求(JSON): %s %s, Body: %s", method, fullURL.String(), string(jsonData))

		} else {
			// --- 默认发送 Form (application/x-www-form-urlencoded) 数据 ---
			contentType = "application/x-www-form-urlencoded"

			// 我们需要将 opt.Body (它是一个 struct) 转换成 map[string]interface{}
			var bodyMap map[string]interface{}
			// 通过一次 json 的序列化和反序列化，可以方便地将任意 struct 转为 map
			bodyBytes, _ := json.Marshal(opt.Body)
			_ = json.Unmarshal(bodyBytes, &bodyMap)

			// 使用 url.Values 来构建表单数据
			formData := url.Values{}
			for key, val := range bodyMap {
				// 将各种类型的值安全地转换为字符串
				formData.Set(key, fmt.Sprintf("%v", val))
			}

			// 将编码后的表单数据作为请求体
			encodedData := formData.Encode()
			bodyReader = bytes.NewBufferString(encodedData)
			c.log.Infof("准备请求(Form): %s %s, Body: %s", method, fullURL.String(), encodedData)
		}
	}

	// --- 3. 创建 http.Request 对象 ---
	req, err := http.NewRequest(method, fullURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("无法创建请求: %w", err)
	}

	// --- 4. 设置所有 Headers ---
	// a. 先设置实例的默认 Header
	for k, v := range c.headers {
		req.Header[k] = v
	}
	// b. 再设置本次请求的 Header（会覆盖默认值）
	if opt.Headers != nil {
		for k, v := range opt.Headers {
			req.Header.Set(k, v)
		}
	}
	// c. 如果有请求体，最后设置 Content-Type
	if opt.Body != nil {
		req.Header.Set("Content-Type", contentType)
	}

	// --- 5. 执行请求拦截器 ---
	for _, interceptor := range c.requestInterceptors {
		if err := interceptor(req); err != nil {
			return nil, fmt.Errorf("request interceptor error: %w", err)
		}
	}

	// --- 6. 发送请求 ---
	c.log.Infof("--> 正在发送请求: %s %s", req.Method, req.URL.String())
	httpRes, err := c.httpClient.Do(req)
	if err != nil {
		// 网络错误、DNS错误、超时等会在这里捕获
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer httpRes.Body.Close()
	c.log.Infof("<-- 收到响应状态码: %d", httpRes.StatusCode)

	// --- 7. 读取响应体 ---
	resBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	c.log.Info("响应体内容:", string(resBody))

	// --- 8. 封装我们自己的 Response ---
	response := &Response{
		StatusCode: httpRes.StatusCode,
		Header:     httpRes.Header,
		Body:       string(resBody),
	}
	c.log.Infof("完整响应: StatusCode=%d, Body=%s", response.StatusCode, response.Body)

	// --- 9. 执行响应拦截器 ---
	for _, interceptor := range c.responseInterceptors {
		if err := interceptor(response); err != nil {
			return nil, fmt.Errorf("response interceptor error: %w", err)
		}
	}

	return response, nil
}

// GetCaptchaImage 获取验证码图片，并返回可直接用于 <img> src 的 Base64 Data URL
func (c *HttpCli) GetCaptchaImage(path string) (string, error) {
	// a. 构建完整的 URL
	fullURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("错误的URL地址: %w", err)
	}
	fullURL, err = fullURL.Parse(path)
	if err != nil {
		return "", fmt.Errorf("错误的请求地址: %w", err)
	}

	// b. 创建一个 GET 请求
	req, err := http.NewRequest(http.MethodGet, fullURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("无法创建验证码请求: %w", err)
	}

	// c. 发送请求。因为我们使用的是带有 cookiejar 的 httpClient，
	//    所以 JSESSIONID 会被自动带上！
	c.log.Infof("--> 正在获取验证码图片: %s", fullURL.String())
	httpRes, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取验证码图片失败: %w", err)
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取验证码失败，状态码: %d", httpRes.StatusCode)
	}

	// d. 读取图片的二进制数据
	imageBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return "", fmt.Errorf("读取验证码图片数据失败: %w", err)
	}

	// e. 将二进制数据进行 Base64 编码
	encodedString := base64.StdEncoding.EncodeToString(imageBytes)

	// f. 获取图片的 MIME 类型 (例如 "image/png")，让 Data URL 更标准
	mimeType := http.DetectContentType(imageBytes)
	if mimeType == "application/octet-stream" { // 如果无法识别，给个默认值
		mimeType = "image/png"
	}

	// g. 拼接成完整的 Data URL 并返回
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encodedString)
	return dataURL, nil
}
