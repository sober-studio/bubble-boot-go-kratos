package server

import (
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http/binding"
	passportV1 "github.com/sober-studio/bubble-boot-go-kratos/api/passport/v1"
	publicV1 "github.com/sober-studio/bubble-boot-go-kratos/api/public/v1"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/auth"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/debug"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/pkg/render"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	app *conf.App,
	public *service.PublicService,
	passport *service.PassportService,
	tokenService auth.TokenService,
	wsSvc *service.WebsocketService,
	logger log.Logger,
) *http.Server {

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			auth.Middleware(tokenService, auth.PathAccessConfigWithPublicList(app.Auth.PublicPaths)),
		),
		http.Filter(debug.Filter),
		http.RequestDecoder(MultipartRequestDecoder),
		http.ResponseEncoder(render.ResponseEncoder),
		http.ErrorEncoder(render.ErrorEncoder),
	}

	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}

	srv := http.NewServer(opts...)

	// 同端口集成点：手动绑定路由
	// 注意：这里用 Handlers.HandleFunc 是绕过 Kratos 的 Proto 解析，直接处理原始 HTTP 请求
	srv.HandleFunc("/ws", wsSvc.WSHandler)

	passportV1.RegisterPassportHTTPServer(srv, passport)
	publicV1.RegisterPublicHTTPServer(srv, public)

	return srv
}

// MultipartRequestDecoder 识别 multipart/form-data 并解析非文件字段
func MultipartRequestDecoder(r *http.Request, v interface{}) error {
	contentType := r.Header.Get("Content-Type")

	// 如果是文件上传
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// 1. 解析 multipart 表单
		// 这一步执行后，非文件字段会被自动填充到 r.Form 中
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return err
		}

		// debug
		fmt.Printf("Form Values: %+v\n", r.Form)

		// 2. 直接传入 r
		// Kratos 会自动从 r.Form 中提取数据并匹配到结构体 v
		if err := binding.BindForm(r, v); err != nil {
			return err
		}
		return nil
	}

	// 如果是普通的 JSON 请求，走默认的解码逻辑
	return http.DefaultRequestDecoder(r, v)
}
