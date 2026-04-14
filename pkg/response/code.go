package response

type Code int

// 常量初始化code值
const (
	SuccessCode   Code = 200
	ParamErrCode  Code = 400
	TokenErrCode  Code = 403
	ServerErrCode Code = 500
)

// map用于存储每个code对应的提示信息
var codeMsgMap = map[Code]string{
	SuccessCode:   "操作成功",
	ParamErrCode:  "参数错误",
	ServerErrCode: "服务端错误",
	TokenErrCode:  "token错误",
}

// 用于获取code对应的提示信息
func (c Code) Msg() string {
	return codeMsgMap[c]
}
