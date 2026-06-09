package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"school-trade/handlers"
	"school-trade/middleware"
	"school-trade/models"
	"school-trade/store"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func initData(dbStore *store.DBStore) {
	db := dbStore.GetDB()
	if db == nil {
		return
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		now := time.Now()
		h1, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		h2, _ := bcrypt.GenerateFromPassword([]byte("alice123"), bcrypt.DefaultCost)
		h3, _ := bcrypt.GenerateFromPassword([]byte("bob123"), bcrypt.DefaultCost)

		users := []models.User{
			{ID: "u-admin", Username: "admin", Password: string(h1), Nickname: "管理员", Phone: "13800000000", Email: "admin@school.edu", Role: "admin", CreatedAt: now, UpdatedAt: now},
			{ID: "u-alice", Username: "alice", Password: string(h2), Nickname: "小艾", Phone: "13800000001", Email: "alice@school.edu", Role: "student", CreatedAt: now, UpdatedAt: now},
			{ID: "u-bob", Username: "bob", Password: string(h3), Nickname: "鲍勃", Phone: "13800000002", Email: "bob@school.edu", Role: "student", CreatedAt: now, UpdatedAt: now},
		}

		for _, u := range users {
			db.Exec("INSERT INTO users (id, username, password, nickname, avatar, phone, email, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				u.ID, u.Username, u.Password, u.Nickname, u.Avatar, u.Phone, u.Email, u.Role, u.CreatedAt, u.UpdatedAt)
		}
	}

	db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if count == 0 {
		now := time.Now()
		imgs := map[string][]string{
			"phone":     {"https://picsum.photos/seed/iphone14/400/400", "https://picsum.photos/seed/iphone14b/400/400", "https://picsum.photos/seed/iphone14c/400/400"},
			"book1":     {"https://picsum.photos/seed/mathbook/400/400", "https://picsum.photos/seed/mathbook2/400/400"},
			"chair":     {"https://picsum.photos/seed/gamingchair/400/400", "https://picsum.photos/seed/gamingchair2/400/400"},
			"racket":    {"https://picsum.photos/seed/badminton/400/400", "https://picsum.photos/seed/badminton2/400/400"},
			"switch":    {"https://picsum.photos/seed/nswitch/400/400", "https://picsum.photos/seed/nswitch2/400/400", "https://picsum.photos/seed/nswitch3/400/400"},
			"book2":     {"https://picsum.photos/seed/englishbook/400/400"},
			"monitor":   {"https://picsum.photos/seed/dellmonitor/400/400", "https://picsum.photos/seed/dellmonitor2/400/400"},
			"airpods":   {"https://picsum.photos/seed/airpodspro/400/400", "https://picsum.photos/seed/airpodspro2/400/400"},
			"camera":    {"https://picsum.photos/seed/canoncam/400/400", "https://picsum.photos/seed/canoncam2/400/400"},
			"bike":      {"https://picsum.photos/seed/bicycle/400/400", "https://picsum.photos/seed/bicycle2/400/400"},
			"guitar":    {"https://picsum.photos/seed/guitar/400/400", "https://picsum.photos/seed/guitar2/400/400"},
			"tablet":    {"https://picsum.photos/seed/ipadpro/400/400", "https://picsum.photos/seed/ipadpro2/400/400"},
			"shoes":     {"https://picsum.photos/seed/nikeshoes/400/400", "https://picsum.photos/seed/nikeshoes2/400/400"},
			"desk":      {"https://picsum.photos/seed/desk/400/400", "https://picsum.photos/seed/desk2/400/400"},
			"lamp":      {"https://picsum.photos/seed/desklamp/400/400"},
			"bag":       {"https://picsum.photos/seed/backpack/400/400", "https://picsum.photos/seed/backpack2/400/400"},
			"headphone": {"https://picsum.photos/seed/sonyheadphone/400/400", "https://picsum.photos/seed/sonyheadphone2/400/400"},
		}
		products := []models.Product{
			{ID: "p-001", Title: "九成新 iPhone 14 128G 星光色", Description: "用了半年，换了新手机所以出掉。无磕碰，屏幕贴膜完好，配件齐全带原装充电线和包装盒。支持当面验机，可小刀。", Category: "electronics", Price: 3800, OriPrice: 5999, Images: imgs["phone"], Condition: "like_new", Campus: "main", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 2 * time.Second), UpdatedAt: now.Add(-86400 * 2 * time.Second)},
			{ID: "p-002", Title: "高等数学第七版上下册全套", Description: "考研复习买的，基本全新，只有上册前两章有少量笔记。课后习题答案齐全，还送同济版线代教材。", Category: "books", Price: 35, OriPrice: 86, Images: imgs["book1"], Condition: "like_new", Campus: "east", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 3 * time.Second), UpdatedAt: now.Add(-86400 * 3 * time.Second)},
			{ID: "p-003", Title: "电竞椅 傲风电竞椅 粉色", Description: "买了新的工学椅所以出掉。坐垫软弹，靠背可放倒165°，扶手可升降。粉色款很少女很好看，送腰靠和头枕。", Category: "furniture", Price: 450, OriPrice: 1299, Images: imgs["chair"], Condition: "good", Campus: "main", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 1 * time.Second), UpdatedAt: now.Add(-86400 * 1 * time.Second)},
			{ID: "p-004", Title: "YONEX 天斧99 羽毛球拍", Description: "正品日版天斧99，4U5规格。拉了26磅BG80线，进攻型球拍，杀球很爽。拍框无内伤，漆水完好轻微使用痕迹。送拍套。", Category: "sports", Price: 620, OriPrice: 1280, Images: imgs["racket"], Condition: "good", Campus: "west", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 5 * time.Second), UpdatedAt: now.Add(-86400 * 5 * time.Second)},
			{ID: "p-005", Title: "Nintendo Switch OLED 白色", Description: "日版续航版，屏幕完美无划痕无坏点。箱说全，送收纳包、钢化膜和摇杆帽。Joy-Con无漂移，平时很爱惜。", Category: "entertainment", Price: 1500, OriPrice: 2599, Images: imgs["switch"], Condition: "like_new", Campus: "south", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 1 * time.Second), UpdatedAt: now.Add(-86400 * 1 * time.Second)},
			{ID: "p-006", Title: "考研英语真题 2024张剑黄皮书", Description: "只做了前3年真题，后面全新空白。还送王江涛作文预测和手写版作文模板笔记，英语一适用。", Category: "books", Price: 20, OriPrice: 69, Images: imgs["book2"], Condition: "fair", Campus: "east", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 7 * time.Second), UpdatedAt: now.Add(-86400 * 7 * time.Second)},
			{ID: "p-007", Title: "戴尔显示器 U2419H 24寸 IPS", Description: "1080P IPS面板，色彩准确适合设计/修图/编程。接口齐全：HDMI+DP+USB Hub。边框超窄，支架可旋转升降。", Category: "electronics", Price: 750, OriPrice: 1599, Images: imgs["monitor"], Condition: "good", Campus: "north", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 4 * time.Second), UpdatedAt: now.Add(-86400 * 4 * time.Second)},
			{ID: "p-008", Title: "AirPods Pro 2代 在保", Description: "去年11月购入，在保到今年11月。降噪通透都很给力，续航正常。换了颜色所以出掉，送全新耳塞一套。", Category: "electronics", Price: 900, OriPrice: 1899, Images: imgs["airpods"], Condition: "like_new", Campus: "main", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 1 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 1 / 2 * time.Second)},
			{ID: "p-009", Title: "佳能 EOS 200D II 单反套机", Description: "入门单反神器，带18-55mm STM镜头。快门数不到3000，几乎全新。翻转触摸屏，自带美颜，适合拍人像和vlog。", Category: "electronics", Price: 2600, OriPrice: 4299, Images: imgs["camera"], Condition: "like_new", Campus: "main", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 5 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 5 / 2 * time.Second)},
			{ID: "p-010", Title: "捷安特 ATX 860 山地车", Description: "27.5寸轮径，禧玛诺27速变速。前后油碟刹车，前叉可锁死。买了一年多骑了不到500公里，送车锁和水壶架。", Category: "sports", Price: 1200, OriPrice: 2598, Images: imgs["bike"], Condition: "good", Campus: "west", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 7 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 7 / 2 * time.Second)},
			{ID: "p-011", Title: "雅马哈 F310 民谣吉他", Description: "入门经典款，云杉面板音色温暖。带琴包、调音器、备用弦和拨片。只弹了几个月，后来没时间学了。", Category: "entertainment", Price: 450, OriPrice: 899, Images: imgs["guitar"], Condition: "like_new", Campus: "east", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 6 * time.Second), UpdatedAt: now.Add(-86400 * 6 * time.Second)},
			{ID: "p-012", Title: "iPad Pro 2022 M2芯片 11寸", Description: "深空灰色，128G WiFi版。M2芯片性能强劲，做笔记画图都很流畅。屏幕无划痕，电池健康92%。送二代笔。", Category: "electronics", Price: 4800, OriPrice: 6799, Images: imgs["tablet"], Condition: "like_new", Campus: "south", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 3 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 3 / 2 * time.Second)},
			{ID: "p-013", Title: "Nike Air Force 1 纯白 42码", Description: "正品Nike AF1，只穿了3次几乎全新。鞋底无磨损，鞋面无折痕。买了发现不太搭自己风格所以出。", Category: "clothing", Price: 380, OriPrice: 799, Images: imgs["shoes"], Condition: "like_new", Campus: "main", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 9 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 9 / 2 * time.Second)},
			{ID: "p-014", Title: "简约电脑桌 书桌 学习桌", Description: "120×60cm，白色简约风格。钢架结构稳固不晃，桌面防水耐磨。搬家换了大桌子这个就闲置了，9成新。", Category: "furniture", Price: 85, OriPrice: 259, Images: imgs["desk"], Condition: "good", Campus: "north", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 8 * time.Second), UpdatedAt: now.Add(-86400 * 8 * time.Second)},
			{ID: "p-015", Title: "小米台灯 Pro 护眼 LED", Description: "国AA级照度，无频闪无蓝光危害。支持色温/亮度无极调节，米家APP遥控。底座有点磕碰痕迹不影响使用。", Category: "furniture", Price: 55, OriPrice: 149, Images: imgs["lamp"], Condition: "good", Campus: "east", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 10 * time.Second), UpdatedAt: now.Add(-86400 * 10 * time.Second)},
			{ID: "p-016", Title: "Herschel 双肩包 Little America", Description: "经典款25L容量，放15寸笔记本无压力。深蓝色，防水面料。背了不到半年，没有破损，拉链顺滑。", Category: "clothing", Price: 180, OriPrice: 598, Images: imgs["bag"], Condition: "good", Campus: "main", SellerID: "u-bob", SellerName: "鲍勃", Status: "selling", CreatedAt: now.Add(-86400 * 11 / 2 * time.Second), UpdatedAt: now.Add(-86400 * 11 / 2 * time.Second)},
			{ID: "p-017", Title: "Sony WH-1000XM5 降噪耳机", Description: "顶级降噪旗舰，音质通透。佩戴舒适不夹头，续航30小时。箱说全带飞机转换头和Type-C充电线。", Category: "electronics", Price: 1350, OriPrice: 2499, Images: imgs["headphone"], Condition: "like_new", Campus: "south", SellerID: "u-alice", SellerName: "小艾", Status: "selling", CreatedAt: now.Add(-86400 * 14 / 5 * time.Second), UpdatedAt: now.Add(-86400 * 14 / 5 * time.Second)},
		}

		for _, p := range products {
			imagesJSON, _ := json.Marshal(p.Images)
			db.Exec(
				"INSERT INTO products (id, title, description, category, price, ori_price, images, cond, campus, seller_id, seller_name, status, view_count, like_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				p.ID, p.Title, p.Description, p.Category, p.Price, p.OriPrice, string(imagesJSON),
				p.Condition, p.Campus, p.SellerID, p.SellerName, p.Status, p.ViewCount, p.LikeCount, p.CreatedAt, p.UpdatedAt,
			)
		}
	}
}

func main() {
	execDir, _ := os.Getwd()
	dataDir := filepath.Join(execDir, "data")
	os.MkdirAll(dataDir, 0755)
	cfgFile := filepath.Join(dataDir, "db-config.json")

	dbStore := store.NewDBStore(cfgFile)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Next()
	})

	// 健康检查接口（云部署必备）
	r.GET("/health", func(c *gin.Context) {
		db := dbStore.GetDB()
		dbOK := db != nil && db.Ping() == nil
		status := "ok"
		if !dbOK {
			status = "degraded"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   status,
			"database": dbOK,
			"time":     time.Now().Format(time.RFC3339),
		})
	})

	api := r.Group("/api")
	{
		// 数据库配置接口（始终可用）
		dbConfigHandler := handlers.NewDBConfigHandler(dbStore, cfgFile)
		api.GET("/db-config", dbConfigHandler.GetConfig)
		api.PUT("/db-config", dbConfigHandler.UpdateConfig)

		// 业务接口
		initData(dbStore)

		authHandler := handlers.NewAuthHandler(dbStore)
		productHandler := handlers.NewProductHandler(dbStore)
		orderHandler := handlers.NewOrderHandler(dbStore)
		socialHandler := handlers.NewSocialHandler(dbStore)
		chatHandler := handlers.NewChatHandler(dbStore)

		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)
		api.GET("/verify-token", authHandler.VerifyToken)

		api.GET("/products", productHandler.List)
		api.GET("/products/:id", productHandler.Get)

		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			auth.GET("/user/me", authHandler.GetCurrentUser)
			auth.PUT("/user/profile", authHandler.UpdateProfile)

			auth.GET("/products/my", productHandler.MyProducts)
			auth.POST("/products", productHandler.Create)
			auth.POST("/upload", productHandler.UploadImage)
			auth.PUT("/products/:id", productHandler.Update)
			auth.DELETE("/products/:id", productHandler.Delete)

			auth.POST("/orders", orderHandler.Create)
			auth.GET("/orders", orderHandler.MyOrders)
			auth.GET("/orders/:id", orderHandler.Get)
			auth.PUT("/orders/:id/status", orderHandler.UpdateStatus)

			// 购物车
			auth.GET("/cart", socialHandler.CartList)
			auth.POST("/cart", socialHandler.CartAdd)
			auth.PUT("/cart/:id", socialHandler.CartUpdate)
			auth.DELETE("/cart/:id", socialHandler.CartDelete)
			auth.DELETE("/cart", socialHandler.CartDelete)

			// 收藏
			auth.GET("/favorites", socialHandler.FavoriteList)
			auth.POST("/favorites", socialHandler.FavoriteToggle)
			auth.GET("/favorites/check", socialHandler.FavoriteCheck)

			// 点赞
			auth.POST("/products/:id/like", socialHandler.LikeToggle)

			// 历史记录
			auth.GET("/history", socialHandler.HistoryList)
			auth.POST("/history", socialHandler.HistoryAdd)

			// 聊天
			auth.GET("/chat/:orderId", chatHandler.ChatHistory)
			auth.GET("/chat/ws/:orderId", chatHandler.ChatWS)
		}

		// 管理员接口
		adminHandler := handlers.NewAdminHandler(dbStore)
		admin := api.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/tables", adminHandler.ListTables)
			admin.GET("/tables/:table", adminHandler.ListRows)
			admin.GET("/tables/:table/:id", adminHandler.GetRow)
			admin.POST("/tables/:table", adminHandler.CreateRow)
			admin.PUT("/tables/:table/:id", adminHandler.UpdateRow)
			admin.DELETE("/tables/:table/:id", adminHandler.DeleteRow)
		}
	}

	frontendDir := filepath.Join(execDir, "..", "frontend")
	r.Static("/css", filepath.Join(frontendDir, "css"))
	r.Static("/js", filepath.Join(frontendDir, "js"))
	r.Static("/pages", filepath.Join(frontendDir, "pages"))
	r.Static("/resources", filepath.Join(frontendDir, "resources"))
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(frontendDir, "index.html"))
	})
	r.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(frontendDir, "index.html"))
	})

	// 端口配置：环境变量 PORT 或默认 28080
	port := os.Getenv("PORT")
	if port == "" {
		port = "28080"
	}
	addr := "0.0.0.0:" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	fmt.Println("========================================")
	fmt.Println("  学校二手交易平台 - 后端服务")
	fmt.Println("  SSO 统一认证已就绪")
	fmt.Println("  运行地址: http://0.0.0.0:" + port)
	fmt.Println("  健康检查: http://0.0.0.0:" + port + "/health")
	fmt.Println("========================================")

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		fmt.Println("\n正在关闭服务...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("服务启动失败: ", err)
	}
	fmt.Println("服务已关闭")
}
