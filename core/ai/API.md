---
title: 默认模块
language_tabs:
  - shell: Shell
  - http: HTTP
  - javascript: JavaScript
  - ruby: Ruby
  - python: Python
  - php: PHP
  - java: Java
  - go: Go
toc_footers: []
includes: []
search: true
code_clipboard: true
highlight_theme: darkula
headingLevel: 2
generator: "@tarslib/widdershins v4.0.30"

---

# 默认模块

Base URLs:

* <a href="http://localhost:11434/api/">正式环境: http://localhost:11434/api/</a>

# Authentication

# 生成补全

## POST 卸载模型

POST /api/generate

If an empty prompt is provided and the `keep_alive` parameter is set to `0`, a model will be unloaded from memory.

> Body 请求参数

```json
{
  "model": "llama3.2",
  "keep_alive": 0
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |none|
|» keep_alive|body|integer| 是 |none|

#### 枚举值

|属性|值|
|---|---|
|» model|llama3.2|

> 返回示例

> 200 Response

```json
{
  "model": "llama3.2",
  "created_at": "2024-09-12T03:54:03.516566Z",
  "response": "",
  "done": true,
  "done_reason": "unload"
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A single JSON object is returned:|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» model|string|true|none||none|
|» created_at|string|true|none||none|
|» response|string|true|none||none|
|» done|boolean|true|none||none|
|» done_reason|string|true|none||none|

# 生成对话补全

## POST 卸载模型

POST /api/chat

If the messages array is empty and the `keep_alive` parameter is set to `0`, a model will be unloaded from memory.

> Body 请求参数

```json
{
  "model": "llama3.2",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "keep_alive": 60
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |none|
|» messages|body|[object]| 是 |none|
|» keep_alive|body|integer| 是 |none|

#### 枚举值

|属性|值|
|---|---|
|» model|llama3.2|

> 返回示例

> 200 Response

```json
{
  "model": "llama3.2",
  "created_at": "2024-09-12T21:33:17.547535Z",
  "message": {
    "role": "assistant",
    "content": ""
  },
  "done_reason": "unload",
  "done": true
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A single JSON object is returned:|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» model|string|true|none||none|
|» created_at|string|true|none||none|
|» message|object|true|none||none|
|»» role|string|true|none||none|
|»» content|string|true|none||none|
|» done_reason|string|true|none||none|
|» done|boolean|true|none||none|

# 创建模型

## POST 从 Safetensors 目录创建模型

POST /api/create

The `files` parameter should include a dictionary of files for the safetensors model which includes the file names and SHA256 digest of each file. Use [/api/blobs/:digest](https://github.com/ollama/ollama/blob/main/docs/api.md#push-a-blob) to first push each of the files to the server before calling this API. Files will remain in the cache until the Ollama server is restarted.

> Body 请求参数

```json
{
  "model": "bert-base-chinese",
  "files": {
    "config.json": "a1b2c3d4e5f6",
    "generation_config.json": "b2c3d4e5f6g7",
    "special_tokens_map.json": "c3d4e5f6g7h8",
    "tokenizer.json": "d4e5f6g7h8i9",
    "tokenizer_config.json": "e5f6g7h8i9j0",
    "model.safetensors": "f6g7h8i9j0k1"
  }
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |模型名称|
|» files|body|object| 是 |none|
|»» config.json|body|string| 是 |配置文件哈希值|
|»» generation_config.json|body|string| 是 |生成配置文件哈希值|
|»» special_tokens_map.json|body|string| 是 |特殊token映射文件哈希值|
|»» tokenizer.json|body|string| 是 |分词器文件哈希值|
|»» tokenizer_config.json|body|string| 是 |分词器配置文件哈希值|
|»» model.safetensors|body|string| 是 |模型权重文件哈希值|

> 返回示例

> 200 Response

```json
"{\"status\":\"converting model\"}\r\n{\"status\":\"creating new layer sha256:05ca5b813af4a53d2c2922933936e398958855c44ee534858fcfd830940618b6\"}\r\n{\"status\":\"using autodetected template llama3-instruct\"}\r\n{\"status\":\"using existing layer sha256:56bb8bd477a519ffa694fc449c2413c6f0e1d3b1c88fa7e3c9d88d3ae49d4dcb\"}\r\n{\"status\":\"writing manifest\"}\r\n{\"status\":\"success\"}"
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A stream of JSON objects is returned:|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» status|string|true|none||none|

# 列出本地模型

## GET 示例

GET /api/tags

> 返回示例

> 200 Response

```json
{
  "models": [
    {
      "name": "codellama:13b",
      "modified_at": "2023-11-04T14:56:49.277302595-07:00",
      "size": 7365960935,
      "digest": "9f438cb9cd581fc025612d27f7c1a6669ff83a8bb0ed86c94fcf4c5440555697",
      "details": {
        "format": "gguf",
        "family": "llama",
        "families": null,
        "parameter_size": "13B",
        "quantization_level": "Q4_0"
      }
    },
    {
      "name": "llama3:latest",
      "modified_at": "2023-12-07T09:32:18.757212583-08:00",
      "size": 3825819519,
      "digest": "fe938a131f40e6f6d40083c9f0f430a515233eb2edaa6d72eb85c50d64f2300e",
      "details": {
        "format": "gguf",
        "family": "llama",
        "families": null,
        "parameter_size": "7B",
        "quantization_level": "Q4_0"
      }
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|A single JSON object will be returned.|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» models|[object]|true|none||none|
|»» name|string|true|none||none|
|»» modified_at|string|true|none||none|
|»» size|integer|true|none||none|
|»» digest|string|true|none||none|
|»» details|object|true|none||none|
|»»» format|string|true|none||none|
|»»» family|string|true|none||none|
|»»» families|null|true|none||none|
|»»» parameter_size|string|true|none||none|
|»»» quantization_level|string|true|none||none|

# 显示模型详情

## POST 示例

POST /api/show

> Body 请求参数

```json
{
  "model": "llama3.2"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |none|

> 返回示例

> 200 Response

```json
{
  "modelfile": "# Modelfile generated by \"ollama show\"\n# To build a new Modelfile based on this one, replace the FROM line with:\n# FROM llava:latest\n\nFROM /Users/matt/.ollama/models/blobs/sha256:200765e1283640ffbd013184bf496e261032fa75b99498a9613be4e94d63ad52\nTEMPLATE \"\"\"{{ .System }}\nUSER: {{ .Prompt }}\nASSISTANT: \"\"\"\nPARAMETER num_ctx 4096\nPARAMETER stop \"</s>\"\nPARAMETER stop \"USER:\"\nPARAMETER stop \"ASSISTANT:\"",
  "parameters": "num_keep                       24\nstop                           \"<|start_header_id|>\"\nstop                           \"<|end_header_id|>\"\nstop                           \"<|eot_id|>\"",
  "template": "{{ if .System }}<|start_header_id|>system<|end_header_id|>\n\n{{ .System }}<|eot_id|>{{ end }}{{ if .Prompt }}<|start_header_id|>user<|end_header_id|>\n\n{{ .Prompt }}<|eot_id|>{{ end }}<|start_header_id|>assistant<|end_header_id|>\n\n{{ .Response }}<|eot_id|>",
  "details": {
    "parent_model": "",
    "format": "gguf",
    "family": "llama",
    "families": [
      "llama"
    ],
    "parameter_size": "8.0B",
    "quantization_level": "Q4_0"
  },
  "model_info": {
    "general.architecture": "llama",
    "general.file_type": 2,
    "general.parameter_count": 8030261248,
    "general.quantization_version": 2,
    "llama.attention.head_count": 32,
    "llama.attention.head_count_kv": 8,
    "llama.attention.layer_norm_rms_epsilon": 0.00001,
    "llama.block_count": 32,
    "llama.context_length": 8192,
    "llama.embedding_length": 4096,
    "llama.feed_forward_length": 14336,
    "llama.rope.dimension_count": 128,
    "llama.rope.freq_base": 500000,
    "llama.vocab_size": 128256,
    "tokenizer.ggml.bos_token_id": 128000,
    "tokenizer.ggml.eos_token_id": 128009,
    "tokenizer.ggml.merges": [],
    "tokenizer.ggml.model": "gpt2",
    "tokenizer.ggml.pre": "llama-bpe",
    "tokenizer.ggml.token_type": [],
    "tokenizer.ggml.tokens": []
  }
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» modelfile|string|true|none||none|
|» parameters|string|true|none||none|
|» template|string|true|none||none|
|» details|object|true|none||none|
|»» parent_model|string|true|none||none|
|»» format|string|true|none||none|
|»» family|string|true|none||none|
|»» families|[string]|true|none||none|
|»» parameter_size|string|true|none||none|
|»» quantization_level|string|true|none||none|
|» model_info|object|true|none||none|
|»» general.architecture|string|true|none||none|
|»» general.file_type|integer|true|none||none|
|»» general.parameter_count|integer|true|none||none|
|»» general.quantization_version|integer|true|none||none|
|»» llama.attention.head_count|integer|true|none||none|
|»» llama.attention.head_count_kv|integer|true|none||none|
|»» llama.attention.layer_norm_rms_epsilon|number|true|none||none|
|»» llama.block_count|integer|true|none||none|
|»» llama.context_length|integer|true|none||none|
|»» llama.embedding_length|integer|true|none||none|
|»» llama.feed_forward_length|integer|true|none||none|
|»» llama.rope.dimension_count|integer|true|none||none|
|»» llama.rope.freq_base|integer|true|none||none|
|»» llama.vocab_size|integer|true|none||none|
|»» tokenizer.ggml.bos_token_id|integer|true|none||none|
|»» tokenizer.ggml.eos_token_id|integer|true|none||none|
|»» tokenizer.ggml.merges|[string]|true|none||populates if `verbose=true`|
|»» tokenizer.ggml.model|string|true|none||none|
|»» tokenizer.ggml.pre|string|true|none||none|
|»» tokenizer.ggml.token_type|[string]|true|none||populates if `verbose=true`|
|»» tokenizer.ggml.tokens|[string]|true|none||populates if `verbose=true`|

# 复制模型

## POST 示例

POST /api/copy

> Body 请求参数

```json
{
  "source": "llama3.2",
  "destination": "llama3-backup"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» source|body|string| 否 |none|
|» destination|body|string| 否 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 删除模型

## DELETE 示例

DELETE /api/delete

> Body 请求参数

```json
{
  "model": "llama3:13b"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 拉取模型

## POST 示例

POST /api/pull

> Body 请求参数

```json
{
  "model": "llama3.2"
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

# 生成嵌入向量

## POST 多输入请求（Multiple Input）

POST /api/embed

> Body 请求参数

```json
{
  "model": "gpt-4",
  "input": [
    "今天天气怎么样？",
    "北京现在的温度是多少？"
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |使用的模型名称|
|» input|body|[string]| 是 |输入的查询问题列表|

> 返回示例

> 200 Response

```json
{
  "model": "all-minilm",
  "embeddings": [
    [
      0.010071029,
      -0.0017594862,
      0.05007221,
      0.04692972,
      0.054916814,
      0.008599704,
      0.105441414,
      -0.025878139,
      0.12958129,
      0.031952348
    ],
    [
      -0.0098027075,
      0.06042469,
      0.025257962,
      -0.006364387,
      0.07272725,
      0.017194884,
      0.09032035,
      -0.051705178,
      0.09951512,
      0.09072481
    ]
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» model|string|true|none||none|
|» embeddings|[array]|true|none||none|

# 列出运行中模型

## POST 示例

POST /api/ps

> 返回示例

> 200 Response

```json
{
  "models": [
    {
      "name": "mistral:latest",
      "model": "mistral:latest",
      "size": 5137025024,
      "digest": "2ae6f6dd7a3dd734790bbbf58b8909a606e0e7e97e94b7604e0aa7ae4490e6d8",
      "details": {
        "parent_model": "",
        "format": "gguf",
        "family": "llama",
        "families": [
          "llama"
        ],
        "parameter_size": "7.2B",
        "quantization_level": "Q4_0"
      },
      "expires_at": "2024-06-04T14:38:31.83753-07:00",
      "size_vram": 5137025024
    }
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» models|[object]|true|none||none|
|»» name|string|false|none||none|
|»» model|string|false|none||none|
|»» size|integer|false|none||none|
|»» digest|string|false|none||none|
|»» details|object|false|none||none|
|»»» parent_model|string|true|none||none|
|»»» format|string|true|none||none|
|»»» family|string|true|none||none|
|»»» families|[string]|true|none||none|
|»»» parameter_size|string|true|none||none|
|»»» quantization_level|string|true|none||none|
|»» expires_at|string|false|none||none|
|»» size_vram|integer|false|none||none|

# 生成单个嵌入向量

## POST 示例

POST /api/embeddings

> Body 请求参数

```json
{
  "model": "all-minilm",
  "prompt": "这是一篇关于羊驼的文章..."
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» model|body|string| 是 |使用的模型名称|
|» prompt|body|string| 是 |输入提示文本|

> 返回示例

> 200 Response

```json
{
  "embedding": [
    0.5670403838157654,
    0.009260174818336964,
    0.23178744316101074,
    -0.2916173040866852,
    -0.8924556970596313,
    0.8785552978515625,
    -0.34576427936553955,
    0.5742510557174683,
    -0.04222835972905159,
    -0.137906014919281
  ]
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» embedding|[number]|true|none||none|

# 数据模型

