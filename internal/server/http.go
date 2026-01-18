package server

import (
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
	logger log.Logger,
) *http.Server {

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			auth.Middleware(tokenService, auth.PathAccessConfigWithPublicList(app.Auth.PublicPaths)),
		),
		http.Filter(debug.Filter),
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
	passportV1.RegisterPassportHTTPServer(srv, passport)
	publicV1.RegisterPublicHTTPServer(srv, public)

	return srv
}
