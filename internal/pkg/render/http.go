package render

import (
	"encoding/json"
	"net/http"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/translator"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Reply 统一 JSON 返回体
type Reply struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Debug 仅在开发/测试环境显示
	Debug interface{}     `json:"debug,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// getCodec 辅助函数：获取编码器，处理双返回值，并默认回退到 JSON
func getCodec(r *http.Request) encoding.Codec {
	// 按照你的要求，处理两个返回值的情况
	// 注：在标准 Kratos v2 中通常返回一个，但若你的版本或封装返回两个，
	// 我们可以通过以下方式确保安全，并强制默认 JSON
	codec, ok := httptransport.CodecForRequest(r, "Accept")
	if !ok || codec == nil {
		codec = encoding.GetCodec("json")
	}
	return codec
}

// ResponseEncoder 成功响应的处理
func ResponseEncoder(w http.ResponseWriter, r *http.Request, data interface{}) error {
	res := &Reply{
		Code:    0,
		Message: "success",
	}

	// 如果是非生产环境，尝试从 Context 捞取调试信息
	if debug.IsDebug() {
		if debugInfo, ok := debug.FromContext(r.Context()); ok {
			res.Debug = debugInfo
		}
	}

	marshaller := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true, // 确保包含默认值
	}

	if m, ok := data.(proto.Message); ok {
		// 3. 关键点：先用 protojson 序列化 Data 部分
		dataBytes, err := marshaller.Marshal(m)
		if err != nil {
			return err
		}
		// 将序列化后的字节流存入 RawMessage
		res.Data = dataBytes
	} else {
		// 如果返回的不是 proto 消息（例如已经是 map 或其他），尝试标准序列化
		b, _ := json.Marshal(data)
		res.Data = b
	}

	codec := getCodec(r)
	body, err := codec.Marshal(res)
	if err != nil {
		return err
	}

	// 设置 Header 并在序列化失败时提供保障
	w.Header().Set("Content-Type", "application/"+codec.Name())
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	return err
}

// ErrorEncoder 错误响应的处理
func ErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	// 1. 将原始 error 转换为 Kratos 的 StatusError
	se := errors.FromError(err)

	// 2. 统一翻译逻辑
	msg := se.Message

	// 如果是 Kratos 官方校验中间件返回的错误，其 Reason 通常是 "INVALID_ARGUMENT"
	// 或者 HTTP 状态码是 400
	if se.Reason == "INVALID_ARGUMENT" {
		// 调用我们之前写的旧版插件翻译器
		// 它会通过接口断言识别出字段名和原因，返回中文
		msg = translator.Translate(se)
	}

	// 3. 包装成你要求的统一格式
	res := &Reply{
		Code:    int(se.Code),
		Message: msg,
		Data:    nil, // 错误时 data 为空
	}

	w.Header().Set("Content-Type", "application/json")
	if errors.Is(se, auth.ErrInvalidToken) || errors.Is(se, auth.ErrTokenExpired) {
		w.WriteHeader(http.StatusUnauthorized)
	} else if se.Code >= 500 {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(int(se.Code))
	}
	_ = json.NewEncoder(w).Encode(res)
	log.Error(err)
}
