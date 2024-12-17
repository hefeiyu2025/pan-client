package thunder_browser

import "time"

type ErrResp struct {
	ErrorCode        int64  `json:"error_code"`
	ErrorMsg         string `json:"error"`
	ErrorDescription string `json:"error_description"`
	//	ErrorDetails   interface{} `json:"error_details"`
}

func (e *ErrResp) IsError() bool {
	return e.ErrorCode != 0 || e.ErrorMsg != "" || e.ErrorDescription != ""
}

type CustomTime struct {
	time.Time
}

const timeFormat = time.RFC3339

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == `""` {
		*ct = CustomTime{Time: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}
		return nil
	}

	t, err := time.Parse(`"`+timeFormat+`"`, str)
	if err != nil {
		return err
	}
	*ct = CustomTime{Time: t}
	return nil
}

type UserMeResp struct {
	Sub         string `json:"sub"`
	Name        string `json:"name"`
	Picture     string `json:"picture"`
	PhoneNumber string `json:"phone_number"`
	Providers   []struct {
		Id             string `json:"id"`
		ProviderUserId string `json:"provider_user_id"`
	} `json:"providers"`
	Password string `json:"password"`
	Status   string `json:"status"`
	Group    []struct {
		Id        string    `json:"id"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"group"`
	CreatedAt         time.Time `json:"created_at"`
	PasswordUpdatedAt time.Time `json:"password_updated_at"`
	Id                string    `json:"id"`
	Vips              []struct {
		Id        string    `json:"id"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"vips"`
	VipInfo []struct {
		Register   string `json:"register"`
		Autodeduct string `json:"autodeduct"`
		Daily      string `json:"daily"`
		Expire     string `json:"expire"`
		Grow       string `json:"grow"`
		IsVip      string `json:"is_vip"`
		LastPay    string `json:"last_pay"`
		Level      string `json:"level"`
		PayId      string `json:"pay_id"`
		Remind     string `json:"remind"`
		IsYear     string `json:"is_year"`
		UserVas    string `json:"user_vas"`
		VasType    string `json:"vas_type"`
		VipDetail  []struct {
			IsVIP      int       `json:"IsVIP"`
			VasType    string    `json:"VasType"`
			Start      time.Time `json:"Start"`
			End        time.Time `json:"End"`
			SurplusDay int       `json:"SurplusDay"`
		} `json:"vip_detail"`
		VipIcon struct {
			General string `json:"general"`
			Small   string `json:"small"`
		} `json:"vip_icon"`
		ExpireTime time.Time `json:"expire_time"`
	} `json:"vip_info"`
}

type LogInRequest struct {
	CaptchaToken string `json:"captcha_token"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type TokenResp struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`

	Sub    string `json:"sub"`
	UserID string `json:"user_id"`

	Token string `json:"token"` // "超级保险箱" 访问Token
}

/*
* 验证码Token
**/
type CaptchaTokenRequest struct {
	Action       string            `json:"action"`
	CaptchaToken string            `json:"captcha_token"`
	ClientID     string            `json:"client_id"`
	DeviceID     string            `json:"device_id"`
	Meta         map[string]string `json:"meta"`
	RedirectUri  string            `json:"redirect_uri"`
}

type CaptchaTokenResponse struct {
	CaptchaToken string `json:"captcha_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Url          string `json:"url"`
}

type FileList struct {
	Kind            string  `json:"kind"`
	NextPageToken   string  `json:"next_page_token"`
	Files           []Files `json:"files"`
	Version         string  `json:"version"`
	VersionOutdated bool    `json:"version_outdated"`
	FolderType      int8
}

type Files struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	//UserID         string    `json:"user_id"`
	Size string `json:"size"`
	//Revision       string    `json:"revision"`
	//FileExtension  string    `json:"file_extension"`
	//MimeType       string    `json:"mime_type"`
	//Starred        bool      `json:"starred"`
	WebContentLink string     `json:"web_content_link"`
	CreatedTime    CustomTime `json:"created_time"`
	ModifiedTime   CustomTime `json:"modified_time"`
	IconLink       string     `json:"icon_link"`
	ThumbnailLink  string     `json:"thumbnail_link"`
	Md5Checksum    string     `json:"md5_checksum"`
	Hash           string     `json:"hash"`
	// Links map[string]Link `json:"links"`
	// Phase string          `json:"phase"`
	// Audit struct {
	// 	Status  string `json:"status"`
	// 	Message string `json:"message"`
	// 	Title   string `json:"title"`
	// } `json:"audit"`
	Medias []struct {
		//Category       string `json:"category"`
		//IconLink       string `json:"icon_link"`
		//IsDefault      bool   `json:"is_default"`
		//IsOrigin       bool   `json:"is_origin"`
		//IsVisible      bool   `json:"is_visible"`
		Link Link `json:"link"`
		//MediaID        string `json:"media_id"`
		//MediaName      string `json:"media_name"`
		//NeedMoreQuota  bool   `json:"need_more_quota"`
		//Priority       int    `json:"priority"`
		//RedirectLink   string `json:"redirect_link"`
		//ResolutionName string `json:"resolution_name"`
		// Video          struct {
		// 	AudioCodec string `json:"audio_codec"`
		// 	BitRate    int    `json:"bit_rate"`
		// 	Duration   int    `json:"duration"`
		// 	FrameRate  int    `json:"frame_rate"`
		// 	Height     int    `json:"height"`
		// 	VideoCodec string `json:"video_codec"`
		// 	VideoType  string `json:"video_type"`
		// 	Width      int    `json:"width"`
		// } `json:"video"`
		// VipTypes []string `json:"vip_types"`
	} `json:"medias"`
	Trashed     bool   `json:"trashed"`
	DeleteTime  string `json:"delete_time"`
	OriginalURL string `json:"original_url"`
	//Params            struct{} `json:"params"`
	//OriginalFileIndex int    `json:"original_file_index"`
	Space string `json:"space"`
	//Apps              []interface{} `json:"apps"`
	//Writable   bool   `json:"writable"`
	FolderType string `json:"folder_type"`
	//Collection interface{} `json:"collection"`
	SortName         string     `json:"sort_name"`
	UserModifiedTime CustomTime `json:"user_modified_time"`
	//SpellName         []interface{} `json:"spell_name"`
	//FileCategory      string        `json:"file_category"`
	//Tags              []interface{} `json:"tags"`
	//ReferenceEvents   []interface{} `json:"reference_events"`
	//ReferenceResource interface{}   `json:"reference_resource"`
	//Params0           struct {
	//	PlatformIcon   string `json:"platform_icon"`
	//	SmallThumbnail string `json:"small_thumbnail"`
	//} `json:"params,omitempty"`
}

type Link struct {
	URL    string    `json:"url"`
	Token  string    `json:"token"`
	Expire time.Time `json:"expire"`
	Type   string    `json:"type"`
}

/*
* 上传
**/
type UploadTaskResponse struct {
	UploadType string `json:"upload_type"`

	//UPLOAD_TYPE_FORM
	Form struct {
		Headers    struct{} `json:"headers"`
		Kind       string   `json:"kind"`
		Method     string   `json:"method"`
		MultiParts struct {
			OSSAccessKeyID string `json:"OSSAccessKeyId"`
			Signature      string `json:"Signature"`
			Callback       string `json:"callback"`
			Key            string `json:"key"`
			Policy         string `json:"policy"`
			XUserData      string `json:"x:user_data"`
		} `json:"multi_parts"`
		URL string `json:"url"`
	} `json:"form"`

	//UPLOAD_TYPE_RESUMABLE
	Resumable struct {
		Kind   string `json:"kind"`
		Params struct {
			AccessKeyID     string    `json:"access_key_id"`
			AccessKeySecret string    `json:"access_key_secret"`
			Bucket          string    `json:"bucket"`
			Endpoint        string    `json:"endpoint"`
			Expiration      time.Time `json:"expiration"`
			Key             string    `json:"key"`
			SecurityToken   string    `json:"security_token"`
		} `json:"params"`
		Provider string `json:"provider"`
	} `json:"resumable"`

	File Files `json:"file"`
}

type UploadTaskRequest struct {
	Kind       string `json:"kind"`
	ParentId   string `json:"parent_id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	UploadType string `json:"upload_type"`
	Space      string `json:"space"`
}

type MkdirResponse struct {
	UploadType string      `json:"upload_type"`
	File       *Files      `json:"file"`
	Task       interface{} `json:"task"`
}
