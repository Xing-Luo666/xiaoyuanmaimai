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
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
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
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type Order struct {
	ID           string     `json:"id"`
	ProductID    string     `json:"productId"`
	ProductTitle string     `json:"productTitle"`
	ProductImage string     `json:"productImage"`
	SpecName     string     `json:"specName"`
	Quantity     int        `json:"quantity"`
	BuyerID      string     `json:"buyerId"`
	BuyerName    string     `json:"buyerName"`
	SellerID     string     `json:"sellerId"`
	SellerName   string     `json:"sellerName"`
	Price        float64    `json:"price"`
	Status       string     `json:"status"`
	Message      string     `json:"message"`
	ShippedAt    *time.Time `json:"shippedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type CartItem struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	ProductID    string    `json:"productId"`
	ProductTitle string    `json:"productTitle"`
	ProductImage string    `json:"productImage"`
	SpecName     string    `json:"specName"`
	Price        float64   `json:"price"`
	Quantity     int       `json:"quantity"`
	Selected     bool      `json:"selected"`
	CreatedAt    time.Time `json:"createdAt"`
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
