package debug

import (
	"context"
	"net/http"
)

// Filter 这是一个 http.Filter，它能修改原始的 http.Request
func Filter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 初始化容器 map
		m := make(Info)
		// 2. 将容器放入 Context，并产生一个新的 Context
		ctx := context.WithValue(r.Context(), debugKey{}, m)

		// 3. 【关键】使用 WithContext 将新的 Context 重新绑定回 Request 对象
		// 这样后续的 r.Context() 就能拿到这个 map 了
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
