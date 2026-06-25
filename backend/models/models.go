package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ProductSpec struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	OriPrice float64 `json:"oriPrice"`
	Stock    int     `json:"stock"`
}

type Address struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Phone      string    `json:"phone"`
	Campus     string    `json:"campus"`
	Building   string    `json:"building"`
	DormNumber string    `json:"dormNumber"`
	IsDefault  bool      `json:"isDefault"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Review struct {
	ID             string     `json:"id"`
	OrderID        string     `json:"orderId"`
	ReviewerID     string     `json:"reviewerId"`
	ReviewerName   string     `json:"reviewerName"`
	ReviewerAvatar string     `json:"reviewerAvatar"`
	TargetID       string     `json:"targetId"`  // 被评价方（卖家ID）
	ProductID      string     `json:"productId"` // 评价的商品ID
	ProductTitle   string     `json:"productTitle"`
	ProductImage   string     `json:"productImage"`
	SpecName       string     `json:"specName"`
	Rating         int        `json:"rating"` // 1-10，半星 step：1=0.5星，2=1星...10=5星
	Content        string     `json:"content"`
	AppendContent  string     `json:"appendContent"` // 追评内容
	AppendAt       *time.Time `json:"appendAt"`      // 追评时间
	HasAppend      bool       `json:"hasAppend"`     // 是否已追评
	Images         []string   `json:"images"`        // 评价图片（多张）
	CreatedAt      time.Time  `json:"createdAt"`
}

type Product struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Price       float64       `json:"price"`
	OriPrice    float64       `json:"oriPrice"`
	Images      []string      `json:"images"`
	Specs       []ProductSpec `json:"specs"`
	Condition   string        `json:"condition"`
	Campus      string        `json:"campus"`
	Building    string        `json:"building"`
	SellerID    string        `json:"sellerId"`
	SellerName  string        `json:"sellerName"`
	Status      string        `json:"status"`
	ViewCount   int           `json:"viewCount"`
	LikeCount   int           `json:"likeCount"`
	FavCount    int           `json:"favCount"`
	RatingAvg   float64       `json:"ratingAvg"`   // 评分均分（0-5，无人打分默认5）
	RatingCount int           `json:"ratingCount"` // 评分人数
	Sold30d     int           `json:"sold30d"`     // 近30天销量（滑动窗口）
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type Order struct {
	ID              string     `json:"id"`
	ProductID       string     `json:"productId"`
	ProductTitle    string     `json:"productTitle"`
	ProductImage    string     `json:"productImage"`
	SpecName        string     `json:"specName"`
	Quantity        int        `json:"quantity"`
	BuyerID         string     `json:"buyerId"`
	BuyerName       string     `json:"buyerName"`
	SellerID        string     `json:"sellerId"`
	SellerName      string     `json:"sellerName"`
	Price           float64    `json:"price"`
	Status          string     `json:"status"`
	Message         string     `json:"message"`
	AddressSnapshot string     `json:"addressSnapshot"`
	ShippedAt       *time.Time `json:"shippedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type CartItem struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	ProductID     string    `json:"productId"`
	ProductTitle  string    `json:"productTitle"`
	ProductImage  string    `json:"productImage"`
	SpecName      string    `json:"specName"`
	Price         float64   `json:"price"`
	Quantity      int       `json:"quantity"`
	Selected      bool      `json:"selected"`
	CreatedAt     time.Time `json:"createdAt"`
	ProductStatus string    `json:"productStatus"` // 关联商品的实时状态：selling/sold_out/...
}

type Favorite struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	ProductID    string    `json:"productId"`
	ProductTitle string    `json:"productTitle"`
	ProductImage string    `json:"productImage"`
	Price        float64   `json:"price"`
	CreatedAt    time.Time `json:"createdAt"`
}

type HistoryItem struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	ProductID    string    `json:"productId"`
	ProductTitle string    `json:"productTitle"`
	ProductImage string    `json:"productImage"`
	Price        float64   `json:"price"`
	ViewedAt     time.Time `json:"viewedAt"`
}

type ChatMessage struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"orderId"`
	SenderID   string    `json:"senderId"`
	SenderName string    `json:"senderName"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	Recalled   bool      `json:"recalled"`
	DeletedBy  string    `json:"deletedBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

type SSOToken struct {
	Token     string `json:"token"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expiresAt"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// ShopInfo 卖家店铺信息
type ShopInfo struct {
	SellerID     string  `json:"sellerId"`
	SellerName   string  `json:"sellerName"`
	SellerAvatar string  `json:"sellerAvatar"`
	ShopRating   float64 `json:"shopRating"`   // 店铺综合评分（所有商品评分均分）
	ReviewCount  int     `json:"reviewCount"`  // 评论总数
	ProductCount int     `json:"productCount"` // 在售商品数
	Sold30d      int     `json:"sold30d"`      // 店铺近30天总销量
}

// ReviewWriteRequest 评价创建请求
type ReviewWriteRequest struct {
	Rating  int      `json:"rating"` // 1-10（半星 step）
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

// ReviewAppendRequest 追评请求
type ReviewAppendRequest struct {
	Content string `json:"content"`
}

// AvatarUploadRequest 头像上传响应
type AvatarUploadResponse struct {
	URL string `json:"url"`
}
