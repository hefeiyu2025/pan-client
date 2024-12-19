package thunder_browser

import (
	"fmt"
	"github.com/hefeiyu2025/pan-client/internal"
	"github.com/hefeiyu2025/pan-client/pan"
	"github.com/imroc/req/v3"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func funReturnBySuccess[T any](err error, response *req.Response, errorResult ErrResp, successResult T) (*T, pan.DriverErrorInterface) {
	if err != nil {
		return nil, pan.OnlyError(err)
	}
	if response.IsErrorState() {
		return nil, pan.CodeMsg(int(errorResult.ErrorCode), errorResult.ErrorMsg)
	}
	return &successResult, pan.NoError()
}

// refreshToken 刷新Token
func (tb *ThunderBrowser) refreshToken(refreshToken string) (*TokenResp, pan.DriverErrorInterface) {
	r := tb.sessionClient.R()
	var successResult TokenResp
	var errorResult ErrResp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(&RefreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
	})
	response, err := r.Post(XLUSER_API_URL + "/auth/token")
	tokenResp, e := funReturnBySuccess(err, response, errorResult, successResult)
	if e == nil {
		tb.setTokenResp(tokenResp)
	}
	return tokenResp, e
}

// 刷新验证码token
func (tb *ThunderBrowser) refreshCaptchaToken(action string, metas map[string]string) pan.DriverErrorInterface {
	r := tb.sessionClient.R()
	var successResult CaptchaTokenResponse
	var errorResult ErrResp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(&CaptchaTokenRequest{
		Action:       action,
		CaptchaToken: tb.properties.CaptchaToken,
		ClientID:     ClientID,
		DeviceID:     tb.properties.DeviceID,
		Meta:         metas,
		RedirectUri:  "xlaccsdk01://xunlei.com/callback?state=harbor",
	})
	resp, err := r.Post(XLUSER_API_URL + "/shield/captcha/init")

	result, e := funReturnBySuccess(err, resp, errorResult, successResult)
	if e != nil {
		return e
	}

	if result.Url != "" {
		return pan.OnlyMsg(fmt.Sprintf(`need verify: <a target="_blank" href="%s">Click Here</a>`, result.Url))
	}

	if result.CaptchaToken == "" {
		return pan.OnlyMsg("empty captchaToken")
	}

	tb.properties.CaptchaToken = result.CaptchaToken
	_ = tb.WriteConfig(tb.properties)
	return nil
}

// GetCaptchaSign 获取验证码签名
func (tb *ThunderBrowser) getCaptchaSign() (timestamp, sign string) {
	timestamp = fmt.Sprint(time.Now().UnixMilli())
	str := fmt.Sprint(ClientID, ClientVersion, PackageName, tb.properties.DeviceID, timestamp)
	for _, algorithm := range Algorithms {
		str = internal.Md5HashStr(str + algorithm)
	}
	sign = "1." + str
	return timestamp, sign
}

// refreshCaptchaTokenAtLogin 刷新验证码token(登录后)
func (tb *ThunderBrowser) refreshCaptchaTokenAtLogin(action, userID string) pan.DriverErrorInterface {
	metas := map[string]string{
		"client_version": ClientVersion,
		"package_name":   PackageName,
		"user_id":        userID,
	}
	metas["timestamp"], metas["captcha_sign"] = tb.getCaptchaSign()
	return tb.refreshCaptchaToken(action, metas)
}

// refreshCaptchaTokenInLogin 刷新验证码token(登录时)
func (tb *ThunderBrowser) refreshCaptchaTokenInLogin(action, username string) pan.DriverErrorInterface {
	metas := make(map[string]string)
	if ok, _ := regexp.MatchString(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`, username); ok {
		metas["email"] = username
	} else if len(username) >= 11 && len(username) <= 18 {
		metas["phone_number"] = username
	} else {
		metas["username"] = username
	}
	return tb.refreshCaptchaToken(action, metas)
}

func GetAction(method string, url string) string {
	urlpath := regexp.MustCompile(`://[^/]+((/[^/\s?#]+)*)`).FindStringSubmatch(url)[1]
	return method + ":" + urlpath
}

func (tb *ThunderBrowser) setTokenResp(tokenResp *TokenResp) {
	tb.properties.TokenType = tokenResp.TokenType
	tb.properties.AccessToken = tokenResp.AccessToken
	tb.properties.RefreshToken = tokenResp.RefreshToken
	tb.properties.ExpiresIn = tokenResp.ExpiresIn
	tb.properties.Sub = tokenResp.Sub
	tb.properties.UserID = tokenResp.UserID
}

func (tb *ThunderBrowser) login(username, password string) (*TokenResp, pan.DriverErrorInterface) {
	url := XLUSER_API_URL + "/auth/signin"
	err := tb.refreshCaptchaTokenInLogin(GetAction(http.MethodPost, url), username)
	if err != nil {
		return nil, err
	}
	r := tb.sessionClient.R()
	var successResult TokenResp
	var errorResult ErrResp
	r.SetSuccessResult(&successResult)
	r.SetErrorResult(&errorResult)
	r.SetBody(&LogInRequest{
		CaptchaToken: tb.properties.CaptchaToken,
		Username:     username,
		Password:     password,
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
	})
	response, reqErr := r.Post(url)
	tokenResp, e := funReturnBySuccess(reqErr, response, errorResult, successResult)
	if e == nil {
		tb.setTokenResp(tokenResp)
	}
	return tokenResp, e
}

func (tb *ThunderBrowser) userMe() (*UserMeResp, pan.DriverErrorInterface) {
	var successResult UserMeResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		return r.Get(XLUSER_API_URL + "/user/me")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) rename(fileId string, newName string) (*Files, pan.DriverErrorInterface) {
	var newFile Files
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetPathParam("fileID", fileId)
		r.SetBody(&pan.Json{"name": newName})
		r.SetQueryParams(map[string]string{
			"space": "",
		})
		r.SetSuccessResult(&newFile)
		return r.Patch(API_URL + "/files/{fileID}")
	})
	return &newFile, err
}

func (tb *ThunderBrowser) makeDir(dirName, dirId string) (*MkdirResponse, pan.DriverErrorInterface) {
	parentId := dirId
	if dirId == "0" {
		parentId = ""
	}
	var successResult MkdirResponse
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetBody(pan.Json{
			"kind":      FOLDER,
			"name":      dirName,
			"parent_id": parentId,
			"space":     ThunderDriveSpace,
		})
		return r.Post(API_URL + "/files")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) move(srcIds []string, destId string) pan.DriverErrorInterface {
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetQueryParams(map[string]string{
			"_from": ThunderDriveSpace,
		})
		r.SetBody(pan.Json{
			"to":    pan.Json{"parent_id": destId, "space": ThunderDriveSpace},
			"space": ThunderDriveSpace,
			"ids":   srcIds,
		})
		return r.Post(API_URL + "/files:batchMove")
	})
	return err
}

func (tb *ThunderBrowser) remove(ids []string) pan.DriverErrorInterface {
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetBody(pan.Json{
			"ids":   ids,
			"space": ThunderDriveSpace,
		})
		return r.Post(API_URL + "/files:batchDelete")
	})
	return err
}

func (tb *ThunderBrowser) getLink(id string) (*Files, pan.DriverErrorInterface) {
	var lFile Files
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetPathParam("fileID", id)
		r.SetQueryParams(map[string]string{
			"_magic":         "2021",
			"space":          ThunderDriveSpace,
			"thumbnail_size": "SIZE_LARGE",
			"with":           "url",
		})
		r.SetSuccessResult(&lFile)
		return r.Get(API_URL + "/files/{fileID}")
	})
	if err != nil {
		return nil, err
	}
	return &lFile, err
}

func (tb *ThunderBrowser) getFiles(dirId string) ([]*Files, pan.DriverErrorInterface) {
	parentId := dirId
	if dirId == "0" {
		parentId = ""
	}
	files := make([]*Files, 0)
	var pageToken string
	for {
		var successResult FileList
		_, err := tb.request(func(r *req.Request) (*req.Response, error) {
			r.SetSuccessResult(&successResult)
			r.SetQueryParams(map[string]string{
				"parent_id":      parentId,
				"page_token":     pageToken,
				"space":          ThunderDriveSpace,
				"filters":        `{"trashed":{"eq":false}}`,
				"with":           "url",
				"with_audit":     "true",
				"thumbnail_size": "SIZE_LARGE",
			})
			return r.Get(API_URL + "/files")
		})
		if err != nil {
			return nil, err
		}

		for _, file := range successResult.Files {
			// 解决 "迅雷云盘" 重复出现问题————迅雷后端发送错误
			if file.FolderType == ThunderDriveFolderType && file.ID == "" && file.Space == "" && dirId != "" {
				continue
			}
			files = append(files, file)
		}

		if successResult.NextPageToken == "" {
			break
		}
		pageToken = successResult.NextPageToken
	}
	return files, nil
}

func (tb *ThunderBrowser) uploadTask(body UploadTaskRequest) (*UploadTaskResponse, pan.DriverErrorInterface) {
	var successResult UploadTaskResponse
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetBody(body)
		return r.Post(API_URL + "/files")
	})
	if err != nil {
		return nil, err
	}
	return &successResult, err
}

func (tb *ThunderBrowser) taskInfo(taskId string) (*Task, pan.DriverErrorInterface) {
	var successResult Task
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetPathParams(map[string]string{
			"taskId": taskId,
		})
		return r.Get(API_URL + "/tasks/{taskId}")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) taskQuery(taskQueryReq TaskQueryRequest) ([]*Task, pan.DriverErrorInterface) {

	tasks := make([]*Task, 0)
	var pageToken string
	filters := `{`
	if len(taskQueryReq.Ids) > 0 {
		filters += `"id":{"in":"` + strings.Join(taskQueryReq.Ids, ",") + `"},`
	}
	if len(taskQueryReq.Phases) > 0 {
		filters += `"phase":{"in":"` + strings.Join(taskQueryReq.Phases, ",") + `"},`
	}
	if len(taskQueryReq.Types) > 0 {
		filters += `"type":{"in":"` + strings.Join(taskQueryReq.Types, ",") + `"},`
	}
	filters = strings.TrimRight(filters, ",")
	filters += `}`
	for {
		var successResult TaskQueryResponse
		_, err := tb.request(func(r *req.Request) (*req.Response, error) {
			r.SetSuccessResult(&successResult)
			r.SetQueryParams(map[string]string{
				"page_token":     pageToken,
				"space":          taskQueryReq.Space,
				"filters":        filters,
				"with":           taskQueryReq.With,
				"limit":          strconv.FormatInt(taskQueryReq.Limit, 10),
				"thumbnail_size": "SIZE_SMALL",
			})
			return r.Get(API_URL + "/tasks")
		})
		if err != nil {
			return nil, err
		}

		for _, task := range successResult.Tasks {
			tasks = append(tasks, task)
		}

		if successResult.NextPageToken == "" {
			break
		}
		pageToken = successResult.NextPageToken
	}
	return tasks, nil
}

func (tb *ThunderBrowser) shareList(shareIds ...string) ([]*ShareInfo, pan.DriverErrorInterface) {
	var pageToken string
	filters := `{`
	if len(shareIds) > 0 {
		filters += `"id":{"in":"` + strings.Join(shareIds, ",") + `"},`
	}
	filters = strings.TrimRight(filters, ",")
	filters += `}`
	shareList := make([]*ShareInfo, 0)
	for {
		var successResult ShareListResp
		_, err := tb.request(func(r *req.Request) (*req.Response, error) {
			r.SetSuccessResult(&successResult)
			r.SetQueryParams(map[string]string{
				"page_token":     pageToken,
				"space":          ThunderDriveSpace,
				"filters":        filters,
				"limit":          "100",
				"thumbnail_size": "SIZE_SMALL",
			})
			return r.Get(API_URL + "/share/list")
		})
		if err != nil {
			return nil, err
		}

		for _, share := range successResult.Data {
			shareList = append(shareList, share)
		}

		if successResult.NextPageToken == "" {
			break
		}
		pageToken = successResult.NextPageToken
	}

	return shareList, nil
}

func (tb *ThunderBrowser) createShare(createShareReq CreateShareReq) (*CreateShareResp, pan.DriverErrorInterface) {
	var successResult CreateShareResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetBody(createShareReq)
		return r.Post(API_URL + "/share")
	})
	if err != nil {
		return nil, err
	}
	return &successResult, nil
}

func (tb *ThunderBrowser) deleteShare(shareId string) pan.DriverErrorInterface {
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetBody(map[string]string{
			"share_id": shareId,
			"space":    ThunderDriveSpace,
		})
		return r.Post(API_URL + "/share/delete")
	})
	return err
}

func (tb *ThunderBrowser) getShare(shareDetailReq ShareDetailReq) (*ShareDetailResp, pan.DriverErrorInterface) {
	var successResult ShareDetailResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetQueryParams(map[string]string{
			"share_id":  shareDetailReq.ShareId,
			"pass_code": shareDetailReq.PassCode,
			"limit":     "100",
			"space":     ThunderDriveSpace,
		})
		return r.Get(API_URL + "/share")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) getShareDetail(shareDetailReq ShareDetailReq) (*ShareDetailResp, pan.DriverErrorInterface) {
	var successResult ShareDetailResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetQueryParams(map[string]string{
			"share_id":        shareDetailReq.ShareId,
			"pass_code_token": shareDetailReq.PassCodeToken,
			"parent_id":       shareDetailReq.ParentId,
			"limit":           "100",
			"space":           ThunderDriveSpace,
		})
		return r.Get(API_URL + "/share/detail")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) about() (*AboutResp, pan.DriverErrorInterface) {
	var successResult AboutResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetQueryParams(map[string]string{
			"with_quotas": QuotaCreateOfflineTaskLimit,
			"space":       ThunderDriveSpace,
		})
		return r.Get(API_URL + "/about")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) restore(restoreReq RestoreReq) (*RestoreResp, pan.DriverErrorInterface) {
	var successResult RestoreResp
	_, err := tb.request(func(r *req.Request) (*req.Response, error) {
		r.SetSuccessResult(&successResult)
		r.SetBody(restoreReq)
		return r.Post(API_URL + "/share/restore")
	})
	return &successResult, err
}

func (tb *ThunderBrowser) request(request func(r *req.Request) (*req.Response, error)) (*req.Response, pan.DriverErrorInterface) {
	r := tb.sessionClient.R()
	r.SetHeaders(map[string]string{
		"Authorization":         fmt.Sprint(tb.properties.TokenType, " ", tb.properties.AccessToken),
		"X-Captcha-Token":       tb.properties.CaptchaToken,
		"X-Space-Authorization": "",
	})
	var errResp ErrResp
	r.SetErrorResult(&errResp)
	data, err := request(r)
	if err != nil {
		return nil, pan.OnlyError(err)
	}

	switch errResp.ErrorCode {
	case 0:
		return data, nil
	case 4122, 4121, 10, 16:
		_, err = tb.refreshToken(tb.properties.RefreshToken)
		if err == nil {
			break
		}
		if tb.properties.Username != "" && tb.properties.Password != "" {
			_, err = tb.login(tb.properties.Username, tb.properties.Password)
			if err == nil {
				break
			}
		}
		return nil, pan.OnlyError(err)
	case 9:
		// space_token 获取失败
		//if errResp.ErrorMsg == "space_token_invalid" {
		//	if token, err := xc.GetSafeAccessToken(xc.Token); err != nil {
		//		return nil, err
		//	} else {
		//		xc.SetSpaceTokenResp(token)
		//	}
		//
		//}
		if errResp.ErrorMsg == "captcha_invalid" {
			// 验证码token过期
			if e := tb.refreshCaptchaTokenAtLogin(GetAction(r.Method, r.RawURL), tb.properties.UserID); e != nil {
				return nil, pan.OnlyError(e)
			}
			break
		}
		return nil, pan.CodeMsg(int(errResp.ErrorCode), errResp.ErrorMsg+errResp.ErrorDescription)
	default:
		return nil, pan.CodeMsg(int(errResp.ErrorCode), errResp.ErrorMsg+errResp.ErrorDescription)
	}
	return tb.request(request)
}
