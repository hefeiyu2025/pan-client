package cloudreve

import (
	"github.com/hefeiyu2025/pan-client/client"
	"github.com/imroc/req/v3"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func funReturn(err error, response *req.Response, result Resp) (*Resp, client.DriverErrorInterface) {
	if err != nil {
		return nil, client.OnlyError(err)
	}
	if response.IsErrorState() && result.Code != 0 {
		return nil, client.CodeMsg(result.Code, result.Msg)
	}
	return &result, client.NoError()
}
func funReturnBySuccess[T any](err error, response *req.Response, errorResult Resp, successResult RespData[T]) (*RespData[T], client.DriverErrorInterface) {
	if err != nil {
		return nil, client.OnlyError(err)
	}
	if response.IsErrorState() {
		return nil, client.CodeMsg(errorResult.Code, errorResult.Msg)
	}
	if successResult.Code != 0 {
		return nil, client.CodeMsg(successResult.Code, successResult.Msg)
	}
	return &successResult, client.NoError()
}

func (c *Cloudreve) config() (*RespData[SiteConfig], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[SiteConfig]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	//site/config
	response, err := r.Get("/site/config")
	if err != nil {
		return nil, client.OnlyError(err)
	}
	if response.IsErrorState() {
		return nil, client.CodeMsg(errorResult.Code, errorResult.Msg)
	}
	if successResult.Code != 0 {
		return nil, client.CodeMsg(successResult.Code, successResult.Msg)
	}
	if !successResult.Data.User.Anonymous {
		for _, cookie := range response.Cookies() {
			if cookie.Name == CookieSessionKey {
				c.properties.CloudreveSession = cookie.Value
				c.properties.RefreshTime = time.Now().UnixMilli()
				c.sessionClient.SetCommonCookies(&http.Cookie{Name: CookieSessionKey, Value: c.properties.CloudreveSession})
			}
		}
	} else {
		return nil, client.OnlyMsg("session is expired")
	}
	return &successResult, client.NoError()
}

func (c *Cloudreve) userStorage() (*RespData[Storage], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[Storage]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	// /file/upload
	response, err := r.Get("/user/storage")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

//	func (c *Cloudreve) FileUpload(sessionId string, index int) Resp {
//		return Resp{}
//	}

func (c *Cloudreve) fileUploadGetUploadSession(req CreateUploadSessionReq) (*RespData[UploadCredential], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[UploadCredential]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(req)
	// /file/upload
	response, err := r.Put("/file/upload")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) fileUploadDeleteUploadSession(sessionId string) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	// /file/upload/{sessionId}
	response, err := r.Delete("/file/upload/" + sessionId)
	return funReturn(err, response, result)
}

func (c *Cloudreve) fileUploadDeleteAllUploadSession() (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	// /file/upload
	response, err := r.Delete("/file/upload")
	return funReturn(err, response, result)
}

//func (c *Cloudreve) FilePutContent(path string) {
//
//}

func (c *Cloudreve) fileCreateFile(path string) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(Json{
		"path": path,
	})
	// /file/create
	response, err := r.Post("/file/create")
	return funReturn(err, response, result)
}

func (c *Cloudreve) fileCreateDownloadSession(id string) (*RespData[string], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[string]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	// /file/download
	response, err := r.Put("/file/download/" + id)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

//func (c *Cloudreve) FilePreview(id string) (string,client.DriverErrorInterface) {
//	r := c.sessionClient.R()
//
//
//	// /file/preview
//	response, err := r.Get("/file/preview/" + id)
//	if err != nil {
//		return "", err
//	}
//
//	return response.String(), nil
//}

//func (c *Cloudreve) FilePreviewText(id string) {
//
//}

func (c *Cloudreve) fileGetSource(req ItemReq) (*RespData[[]Sources], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[[]Sources]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(req)
	// /file/source
	response, err := r.Post("/file/source")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) fileArchive(req ItemReq) (*RespData[string], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[string]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(req)
	// /file/archive
	response, err := r.Post("/file/archive")
	return funReturnBySuccess(err, response, errorResult, successResult)
}
func (c *Cloudreve) createDirectory(path string) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(Json{
		"path": path,
	})
	//directory
	response, err := r.Put("/directory")
	return funReturn(err, response, result)
}

func (c *Cloudreve) listDirectory(path string) (*RespData[ObjectList], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[ObjectList]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	// /directory*path
	response, err := r.Get("/directory" + url.PathEscape(path))
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) objectDelete(req ItemReq) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(req)
	// object
	response, err := r.Delete("/object")
	return funReturn(err, response, result)
}

func (c *Cloudreve) objectMove(req ItemMoveReq) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(req)
	// object
	response, err := r.Patch("/object")
	return funReturn(err, response, result)
}

func (c *Cloudreve) objectCopy(req ItemMoveReq) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(req)
	// /object/copy
	response, err := r.Post("/object/copy")
	return funReturn(err, response, result)
}

func (c *Cloudreve) objectRename(req ItemRenameReq) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	r.SetBody(req)
	// /object/rename
	response, err := r.Post("/object/rename")
	return funReturn(err, response, result)
}

func (c *Cloudreve) objectGetProperty(req ItemPropertyReq) (*RespData[ObjectProps], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var errorResult Resp
	var successResult RespData[ObjectProps]
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetQueryParamsAnyType(Json{
		"is_folder":  req.IsFolder,
		"trace_root": req.TraceRoot,
	})
	// /object/property/{id}
	response, err := r.Get("/object/property/" + req.Id)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareCreateShare(req ShareCreateReq) (*RespData[string], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[string]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(req)
	// /share
	response, err := r.Post("/share")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareListShare() (*RespData[ShareList], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[ShareList]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	// /share
	response, err := r.Get("/share")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareUpdateShare(req ShareUpdateReq) (*RespData[string], client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var successResult RespData[string]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(Json{
		"prop":  req.Prop,
		"value": req.Value,
	})
	// /share
	response, err := r.Patch("/share/" + req.Id)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareDeleteShare(id string) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	// /share
	response, err := r.Delete("/share/" + id)
	return funReturn(err, response, result)
}

func (c *Cloudreve) shareGetShare(id, password string) (*RespData[Share], client.DriverErrorInterface) {
	r := c.defaultClient.R()
	var successResult RespData[Share]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(Json{
		"password": password,
	})
	// /share
	response, err := r.Get("/share/info/" + id)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareGetShareDownload(id, path string) (*RespData[string], client.DriverErrorInterface) {
	r := c.defaultClient.R()
	var successResult RespData[string]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetQueryParamsAnyType(Json{
		"path": path,
	})
	// /share
	response, err := r.Put("/share/download/" + id)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareListSharedFolder(id, path string) (*RespData[ObjectList], client.DriverErrorInterface) {
	r := c.defaultClient.R()
	var successResult RespData[ObjectList]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)

	// /share/list/:id/*path
	response, err := r.Put("/share/list/" + id + url.PathEscape(path))
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) ShareSearchSharedFolder(id, keyword, path string, searchType SearchType) (*RespData[ObjectList], client.DriverErrorInterface) {
	r := c.defaultClient.R()
	var successResult RespData[ObjectList]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetQueryParamsAnyType(Json{
		"path": path,
	})
	// /share/search/:id/:type/:keywords
	response, err := r.Get("/share/search/" + id + "/" + string(searchType) + "/" + keyword)
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) shareSearchShare(req ShareListReq) (*RespData[ShareList], client.DriverErrorInterface) {
	r := c.defaultClient.R()
	var successResult RespData[ShareList]
	var errorResult Resp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetQueryParamsAnyType(Json{
		"page":     req.Page,
		"order_by": req.OrderBy,
		"order":    req.Order,
		"keywords": req.Keywords,
	})
	// /share/search
	response, err := r.Get("/share/search")
	return funReturnBySuccess(err, response, errorResult, successResult)
}

func (c *Cloudreve) oneDriveCallback(sessionId string) (*Resp, client.DriverErrorInterface) {
	r := c.sessionClient.R()
	var result Resp
	r.SetSuccessResult(&result)
	r.SetErrorResult(&result)
	// /callback/onedrive/finish/:sessionID
	response, err := r.SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody("{}").
		Post("/callback/onedrive/finish/" + sessionId)
	return funReturn(err, response, result)
}

// OneDriveUpload 分片上传 返回已上传的字节数和错误信息
func (c *Cloudreve) oneDriveUpload(req OneDriveUploadReq) (int64, client.DriverErrorInterface) {
	uploadedSize := req.UploadedSize

	pr, err := client.NewProcessReader(req.LocalFile, req.ChunkSize, uploadedSize)
	if err != nil {
		return uploadedSize, err
	}
	for {
		startSize, endSize := pr.NextChunk()
		response, reqErr := c.defaultClient.R().SetBody(pr).
			SetContentType("application/octet-stream").
			SetHeader("Content-Length", strconv.FormatInt(endSize-startSize, 10)).
			SetHeader("Content-Range", "bytes "+strconv.FormatInt(startSize, 10)+"-"+strconv.FormatInt(endSize-1, 10)+"/"+strconv.FormatInt(pr.GetTotal(), 10)).
			Put(req.UploadUrl)
		if reqErr != nil {
			return pr.GetUploaded(), client.OnlyError(reqErr)
		}
		if response.IsErrorState() {
			return pr.GetUploaded(), client.OnlyMsg(response.String())
		}

		if pr.IsFinish() {
			break
		}
	}
	pr.Close()
	return pr.GetUploaded(), client.NoError()
}
