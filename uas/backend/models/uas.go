package models

import "time"

// UUser 自然人用户
type UUser struct {
	ID          int64      `json:"id"`
	Phone       string     `json:"phone"`
	Password    string     `json:"-"`
	RealName    string     `json:"realName"`
	IDCardType  int        `json:"idCardType"`
	IDCardNo    string     `json:"idCardNo"` // 已脱敏
	IDCardNoRaw string     `json:"-"`        // 原始值，不返回前端
	AuthLevel   string     `json:"authLevel"`
	Avatar      string     `json:"avatar"`
	Nickname    string     `json:"nickname"`
	Email       string     `json:"email"`
	Status      int        `json:"status"`
	AuditStatus int        `json:"auditStatus"`
	AuditRemark string     `json:"auditRemark"`
	AuditTime   *time.Time `json:"auditTime"`
	CreateTime  time.Time  `json:"createTime"`
	UpdateTime  time.Time  `json:"updateTime"`
}

// UCorpUser 法人用户
type UCorpUser struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Password    string     `json:"-"`
	CorpType    string     `json:"corpType"`
	CorpName    string     `json:"corpName"`
	CreditCode  string     `json:"creditCode"`
	LegalPerson string     `json:"legalPerson"`
	LegalIDCard string     `json:"legalIdCard"`
	AgentName   string     `json:"agentName"`
	AgentIDCard string     `json:"agentIdCard"`
	Phone       string     `json:"phone"`
	Status      int        `json:"status"`
	AuditStatus int        `json:"auditStatus"`
	AuditRemark string     `json:"auditRemark"`
	AuditTime   *time.Time `json:"auditTime"`
	CreateTime  time.Time  `json:"createTime"`
	UpdateTime  time.Time  `json:"updateTime"`
}

// UApp 第三方应用
type UApp struct {
	ID          int64     `json:"id"`
	AppID       string    `json:"appId"`
	AppName     string    `json:"appName"`
	AppType     string    `json:"appType"`
	SM4Secret   string    `json:"sm4Secret"` // 脱敏
	AppSecret   string    `json:"appSecret"` // 脱敏
	RedirectURI string    `json:"redirectUri"`
	Status      int       `json:"status"`
	Description string    `json:"description"`
	CreateTime  time.Time `json:"createTime"`
	UpdateTime  time.Time `json:"updateTime"`
}

// ULoginLog 登录日志
type ULoginLog struct {
	ID          int64     `json:"id"`
	UserID      *int64    `json:"userId"`
	Username    string    `json:"username"`
	LoginType   string    `json:"loginType"`
	LoginIP     string    `json:"loginIp"`
	LoginResult int       `json:"loginResult"`
	FailReason  string    `json:"failReason"`
	UserAgent   string    `json:"userAgent"`
	LoginTime   time.Time `json:"loginTime"`
}

// SysAuditLog 审计日志
type SysAuditLog struct {
	ID          int64     `json:"id"`
	OperName    string    `json:"operName"`
	OperType    string    `json:"operType"`
	OperContent string    `json:"operContent"`
	OperIP      string    `json:"operIp"`
	OperTime    time.Time `json:"operTime"`
}

// SysSmsLog 短信日志
type SysSmsLog struct {
	ID         int64     `json:"id"`
	Phone      string    `json:"phone"`
	Template   string    `json:"template"`
	Content    string    `json:"content"`
	SendResult string    `json:"sendResult"`
	SendTime   time.Time `json:"sendTime"`
}

// UGrant 应用授权
type UGrant struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	UserType   string     `json:"userType"`
	AppID      string     `json:"appId"`
	GrantTime  time.Time  `json:"grantTime"`
	ExpireTime *time.Time `json:"expireTime"`
	Status     int        `json:"status"`
	// 关联字段
	AppName  string `json:"appName,omitempty" db:"-"`
	UserName string `json:"userName,omitempty" db:"-"`
}
