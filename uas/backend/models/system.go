package models

import "time"

// SysUser 系统管理员
type SysUser struct {
	ID         int64     `json:"id"`
	Username   string    `json:"username"`
	Password   string    `json:"-"` // 不返回密码
	Nickname   string    `json:"nickname"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Sex        int       `json:"sex"`
	Avatar     string    `json:"avatar"`
	Status     int       `json:"status"`
	DeptID     *int64    `json:"deptId"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	// 关联（不映射数据库字段）
	RoleIDs []int64  `json:"roleIds,omitempty" db:"-"`
	Roles   []string `json:"roles,omitempty" db:"-"`
}

// SysRole 角色
type SysRole struct {
	ID         int64     `json:"id"`
	RoleName   string    `json:"roleName"`
	RoleKey    string    `json:"roleKey"`
	RoleSort   int       `json:"roleSort"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

// SysMenu 菜单
type SysMenu struct {
	ID         int64     `json:"id"`
	MenuName   string    `json:"menuName"`
	ParentID   int64     `json:"parentId"`
	MenuSort   int       `json:"menuSort"`
	Path       string    `json:"path"`
	Component  string    `json:"component"`
	MenuType   string    `json:"menuType"` // M-目录 C-菜单 F-按钮
	Visible    int       `json:"visible"`
	Perms      string    `json:"perms"`
	Icon       string    `json:"icon"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	Children   []SysMenu `json:"children,omitempty" db:"-"`
}

// SysDept 部门
type SysDept struct {
	ID         int64     `json:"id"`
	ParentID   int64     `json:"parentId"`
	DeptName   string    `json:"deptName"`
	DeptSort   int       `json:"deptSort"`
	Leader     string    `json:"leader"`
	Phone      string    `json:"phone"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	Children   []SysDept `json:"children,omitempty" db:"-"`
}

// SysPost 岗位
type SysPost struct {
	ID         int64     `json:"id"`
	PostCode   string    `json:"postCode"`
	PostName   string    `json:"postName"`
	PostSort   int       `json:"postSort"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

// SysDictType 字典类型
type SysDictType struct {
	ID         int64     `json:"id"`
	DictName   string    `json:"dictName"`
	DictType   string    `json:"dictType"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

// SysDictData 字典数据
type SysDictData struct {
	ID         int64     `json:"id"`
	DictType   string    `json:"dictType"`
	DictLabel  string    `json:"dictLabel"`
	DictValue  string    `json:"dictValue"`
	DictSort   int       `json:"dictSort"`
	CssClass   string    `json:"cssClass"`
	ListClass  string    `json:"listClass"`
	IsDefault  int       `json:"isDefault"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

// SysConfig 参数配置
type SysConfig struct {
	ID          int64     `json:"id"`
	ConfigName  string    `json:"configName"`
	ConfigKey   string    `json:"configKey"`
	ConfigValue string    `json:"configValue"`
	ConfigType  string    `json:"configType"`
	Remark      string    `json:"remark"`
	CreateTime  time.Time `json:"createTime"`
	UpdateTime  time.Time `json:"updateTime"`
}

// SysNotice 通知公告
type SysNotice struct {
	ID            int64     `json:"id"`
	NoticeTitle   string    `json:"noticeTitle"`
	NoticeType    int       `json:"noticeType"`
	NoticeContent string    `json:"noticeContent"`
	Status        int       `json:"status"`
	CreateBy      string    `json:"createBy"`
	CreateTime    time.Time `json:"createTime"`
	UpdateTime    time.Time `json:"updateTime"`
}
