package middleware

import "github.com/gogf/gf/v2/net/ghttp"

// CORSMiddleware 处理CORS跨域请求
// 浏览器交互的过程会先给服务器发送一个OPTIONS请求，询问服务器是否允许跨域请求。
// 如果服务器允许跨域请求，则会返回一个包含CORS相关头信息的响应，浏览器会根据这些头信息决定是否继续发送实际的请求。
// 如果服务器不允许跨域请求，则会返回一个不包含CORS相关头信息的响应，浏览器会阻止实际请求的发送。
// 现阶段本地开发过程中,直接使用默认策略,允许所有跨域请求,后续可根据实际情况进行调整
func CORSMiddleware(r *ghttp.Request) {
	r.Response.CORSDefault()
	if r.Method == "OPTIONS" {
		r.Response.WriteStatus(204)
		return
	}
	r.Middleware.Next() //接着进行后面的访问
}

func ResponseMiddleware(r *ghttp.Request) {
	r.Middleware.Next() //后置中间件,在请求处理完成后再执行校验

	var (
		msg string
		res = r.GetHandlerResponse()
		err = r.GetError()
	)
	if err != nil {
		msg = err.Error()
	} else {
		msg = "OK"
	}
	r.Response.WriteJson(Response{
		Message: msg,
		Data:    res,
	})
}

type Response struct {
	Message string      `json:"message" dc:"消息提示"`
	Data    interface{} `json:"data"    dc:"执行结果"`
}
