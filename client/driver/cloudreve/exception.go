package cloudreve

import "github.com/hefeiyu2025/pan-client/client"

const (
	NOERR   int = 0
	UNKNOWN int = 9999
)

type CloudreveError struct {
	client.DriverError
}

func NoError() client.DriverErrorInterface {
	return nil
}

func OnlyError(error error) *CloudreveError {
	return MsgError("", error)
}

func OnlyCode(code int) *CloudreveError {
	return CodeMsg(code, "")
}

func OnlyMsg(msg string) *CloudreveError {
	return MsgError(msg, nil)
}

func MsgError(msg string, error error) *CloudreveError {
	return MsgErrorData(msg, error, nil)
}

func MsgErrorData(msg string, error error, data interface{}) *CloudreveError {
	return CodeMsgErrorData(UNKNOWN, msg, error, data)
}

func CodeMsgError(code int, msg string, error error) *CloudreveError {
	return CodeMsgErrorData(code, msg, error, nil)
}

func CodeMsg(code int, msg string) *CloudreveError {
	return CodeMsgErrorData(code, msg, nil, nil)
}

func CodeMsgErrorData(code int, msg string, error error, data interface{}) *CloudreveError {
	return &CloudreveError{DriverError: client.DriverError{
		Code: code,
		Msg:  msg,
		Err:  error,
		Data: data,
	}}
}
