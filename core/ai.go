package core

// Message 聊天消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIProvider AI提供者接口
type AIProvider interface {
	// Chat 发送聊天消息
	Chat(model string, messages []Message) (string, error)
	// ChatStream 发送聊天消息并流式返回结果
	ChatStream(model string, messages []Message, callback func(string)) error
	// ListModels 获取模型列表
	ListModels() ([]string, error)
	// Validate 验证连接和配置
	Validate() error
}
