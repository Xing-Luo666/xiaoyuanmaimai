package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:114514@tcp(127.0.0.1:3306)/school_trade?charset=utf8mb4&parseTime=true&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(err)
	}

	// 取前 12 件在售商品
	type P struct {
		ID, Title, Image, SellerID, SellerName string
		Price                                  float64
	}
	rows, err := db.Query(`SELECT id, title, COALESCE((SELECT image_url FROM (SELECT JSON_UNQUOTE(JSON_EXTRACT(images, '$[0]')) AS image_url FROM products p2 WHERE p2.id = products.id) tmp), ''), seller_id, COALESCE(seller_name,''), price FROM products WHERE status='selling' ORDER BY created_at DESC LIMIT 12`)
	if err != nil {
		fmt.Println("查询商品失败:", err)
		return
	}
	var ps []P
	for rows.Next() {
		var p P
		if err := rows.Scan(&p.ID, &p.Title, &p.Image, &p.SellerID, &p.SellerName, &p.Price); err != nil {
			fmt.Println("扫描失败:", err)
			return
		}
		ps = append(ps, p)
	}
	rows.Close()
	fmt.Printf("获取到 %d 件商品\n", len(ps))

	// 买家池
	buyers := []struct{ ID, Name string }{
		{"u-alice", "小艾"},
		{"u-bob", "鲍勃"},
		{"u-carol", "卡罗"},
		{"u-dave", "大卫"},
		{"u-eve", "伊芙"},
	}

	// 评价内容池
	contents := []string{
		"商品和描述一致，物流也很快，卖家态度非常好！",
		"成色不错，性价比高，推荐购买。",
		"包装很用心，商品完好无损，点赞。",
		"比预期还好一些，卖家很耐心解答问题。",
		"物超所值，下次还会光顾。",
		"商品基本符合描述，有一点点小瑕疵但能接受。",
		"发货速度快，商品质量不错。",
		"很满意的一次购物体验，好评！",
		"东西收到了，和图片一样，挺喜欢的。",
		"卖家服务态度好，商品也行，给个好评。",
	}
	ratings := []int{10, 10, 9, 10, 8, 9, 10, 7, 9, 10} // 1-10 分

	now := time.Now()
	orderCnt, reviewCnt := 0, 0

	for i, p := range ps {
		// 每件商品随机生成 2-5 单已完成订单（30 天内）
		orderN := 2 + (i % 4)
		for j := 0; j < orderN; j++ {
			b := buyers[(i+j)%len(buyers)]
			if b.ID == p.SellerID {
				b = buyers[(i+j+1)%len(buyers)]
			}
			orderID := fmt.Sprintf("o-seed-%d-%d", i, j)
			createdAt := now.AddDate(0, 0, -(1 + (i*3+j*2)%25))
			_, err := db.Exec(`INSERT IGNORE INTO orders (id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, address_id, address_snapshot, created_at, updated_at) VALUES (?,?,?,?,'',1,?,?,?,?,?,'completed','','','',?,?)`,
				orderID, p.ID, p.Title, p.Image, b.ID, b.Name, p.SellerID, p.SellerName, p.Price, createdAt, createdAt)
			if err != nil {
				fmt.Printf("插入订单失败 %s: %v\n", orderID, err)
				continue
			}
			orderCnt++

			// 80% 概率给评价（buyer 评价 seller）
			if (i+j)%5 != 0 {
				ridx := (i + j) % len(ratings)
				reviewID := fmt.Sprintf("r-seed-%d-%d", i, j)
				rating := ratings[ridx]
				content := contents[ridx]
				_, err := db.Exec(`INSERT IGNORE INTO reviews (id, order_id, reviewer_id, reviewer_name, target_id, product_id, product_title, product_image, spec_name, rating, content, images, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?, '[]', ?)`,
					reviewID, orderID, b.ID, b.Name, p.SellerID, p.ID, p.Title, p.Image, "", rating, content, createdAt.Add(time.Hour))
				if err != nil {
					fmt.Printf("插入评价失败 %s: %v\n", reviewID, err)
					continue
				}
				reviewCnt++
			}
		}

		// 增加浏览量
		newViews := 5 + (i*7)%80
		_, _ = db.Exec(`UPDATE products SET view_count = view_count + ? WHERE id = ?`, newViews, p.ID)
	}

	fmt.Printf("已插入 %d 单测试订单，%d 条测试评价，并更新浏览量\n", orderCnt, reviewCnt)

	// 验证结果
	var avg float64
	var cnt int
	db.QueryRow("SELECT AVG(rating)/2, COUNT(*) FROM reviews").Scan(&avg, &cnt)
	fmt.Printf("当前全表评分均值: %.2f, 评价总数: %d\n", avg, cnt)

	var soldSum int
	db.QueryRow("SELECT COALESCE(SUM(quantity),0) FROM orders WHERE status='completed' AND created_at >= ?", now.AddDate(0, 0, -30)).Scan(&soldSum)
	fmt.Printf("近30天完成订单总销量: %d\n", soldSum)
}
