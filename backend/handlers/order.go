package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"school-trade/models"
	"school-trade/store"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	Store *store.DBStore
	Chat  *ChatHandler
}

func NewOrderHandler(s *store.DBStore, ch *ChatHandler) *OrderHandler {
	return &OrderHandler{Store: s, Chat: ch}
}

type createOrderReq struct {
	ProductID string               `json:"productId" binding:"required"`
	Message   string               `json:"message"`
	Price     float64              `json:"price"`
	Quantity  int                  `json:"quantity"`
	Spec      string               `json:"spec"`
	Specs     []createOrderReqSpec `json:"specs"`
	AddressID string               `json:"addressId"`
}

type createOrderReqSpec struct {
	SpecID   string  `json:"specId"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func (h *OrderHandler) Create(c *gin.Context) {
	userID := c.GetString("userId")
	username := c.GetString("username")

	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	// 兼容前端 specs 数组格式：从 specs[0] 提取 spec
	if req.Spec == "" && len(req.Specs) > 0 {
		req.Spec = req.Specs[0].SpecID
		if req.Spec == "" {
			req.Spec = req.Specs[0].Name
		}
		if req.Quantity <= 0 {
			req.Quantity = req.Specs[0].Quantity
		}
		if req.Price <= 0 && req.Specs[0].Price > 0 {
			req.Price = req.Specs[0].Price
		}
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	db := h.Store.GetDB()

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "下单失败"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 在事务内用行锁读取商品信息，杜绝竞态
	var title, sellerID, sellerName, status string
	var price float64
	var productImage sql.NullString
	err = tx.QueryRow("SELECT title, seller_id, seller_name, price, status, COALESCE(JSON_UNQUOTE(JSON_EXTRACT(images, '$[0]')), '') FROM products WHERE id = ? FOR UPDATE", req.ProductID).Scan(&title, &sellerID, &sellerName, &price, &status, &productImage)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "商品不存在"})
		return
	}

	if status != "selling" {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该商品已下架或已售出"})
		return
	}
	if sellerID == userID {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不能购买自己的商品"})
		return
	}

	finalPrice := price
	if req.Price > 0 {
		finalPrice = req.Price
	}

	// 如果有收货地址，查询并存储地址快照
	var addressSnapshot string
	if req.AddressID != "" {
		var phone, campus, building, dormNumber string
		err = tx.QueryRow("SELECT phone, campus, building, dorm_number FROM user_addresses WHERE id = ? AND user_id = ?", req.AddressID, userID).Scan(&phone, &campus, &building, &dormNumber)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "收货地址不存在"})
			return
		}
		snap, _ := json.Marshal(map[string]string{
			"phone":      phone,
			"campus":     campus,
			"building":   building,
			"dormNumber": dormNumber,
		})
		addressSnapshot = string(snap)
	}

	now := time.Now()
	order := models.Order{
		ID:              genID("o"),
		ProductID:       req.ProductID,
		ProductTitle:    title,
		ProductImage:    productImage.String,
		SpecName:        req.Spec,
		Quantity:        req.Quantity,
		BuyerID:         userID,
		BuyerName:       username,
		SellerID:        sellerID,
		SellerName:      sellerName,
		Price:           finalPrice * float64(req.Quantity),
		Status:          "pending",
		Message:         req.Message,
		AddressSnapshot: addressSnapshot,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 如果有规格，减少对应规格的库存
	if req.Spec != "" {
		var specsNS sql.NullString
		if err := tx.QueryRow("SELECT specs FROM products WHERE id = ? FOR UPDATE", req.ProductID).Scan(&specsNS); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询商品规格失败"})
			return
		}
		specsJSON := ""
		if specsNS.Valid {
			specsJSON = specsNS.String
		}
		// 商品无规格数据，但用户传了 spec，必须报错（防止超卖）
		if specsJSON == "" || specsJSON == "null" {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "商品规格不存在: " + req.Spec})
			return
		}
		var specs []models.ProductSpec
		if err := json.Unmarshal([]byte(specsJSON), &specs); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "商品规格数据异常"})
			return
		}
		found := false
		for i := range specs {
			if specs[i].Name == req.Spec || specs[i].ID == req.Spec {
				// stock == -1 表示无限库存；stock == 0 表示已售罄
				if specs[i].Stock == 0 {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "该规格已售罄"})
					return
				}
				if specs[i].Stock > 0 && specs[i].Stock < req.Quantity {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "库存不足，剩余 " + fmt.Sprintf("%d", specs[i].Stock) + " 件"})
					return
				}
				if specs[i].Stock > 0 {
					specs[i].Stock -= req.Quantity
				}
				found = true
				break
			}
		}
		// 关键修复：规格未找到时必须报错，否则订单创建但库存未扣，造成超卖
		if !found {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "商品规格不存在: " + req.Spec})
			return
		}
		newSpecsJSON, _ := json.Marshal(specs)
		if _, err := tx.Exec("UPDATE products SET specs = ? WHERE id = ?", string(newSpecsJSON), req.ProductID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品规格失败"})
			return
		}

		// 检查是否所有规格库存都为0（排除无限库存 -1）
		allSoldOut := true
		for _, s := range specs {
			if s.Stock == -1 || s.Stock > 0 {
				allSoldOut = false
				break
			}
		}
		if allSoldOut {
			if _, err := tx.Exec("UPDATE products SET status = 'sold_out' WHERE id = ?", req.ProductID); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
				return
			}
		}
	}

	_, err = tx.Exec(
		"INSERT INTO orders (id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, address_id, address_snapshot, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		order.ID, order.ProductID, order.ProductTitle, order.ProductImage, order.SpecName, order.Quantity, order.BuyerID, order.BuyerName, order.SellerID, order.SellerName, order.Price, order.Status, order.Message, req.AddressID, order.AddressSnapshot, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "创建订单失败"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "下单失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "下单成功", Data: order})

	// 发送系统消息通知卖家
	if h.Chat != nil {
		h.Chat.SendSystemMsg(order.ID, userID, username, "bought", title, productImage.String, req.Spec, order.Price, order.Quantity)
	}
}

func (h *OrderHandler) MyOrders(c *gin.Context) {
	userID := c.GetString("userId")
	role := c.DefaultQuery("role", "buyer")
	tab := c.DefaultQuery("tab", "all") // all/pending_pay/pending_ship/pending_recv/refund/pending_review

	db := h.Store.GetDB()

	// tab → status 过滤条件
	// 淘宝6板块：全部/待付款/待发货/待收货/"退款/售后"/待评价
	// 本系统无支付环节，"待付款"映射为 pending（待处理）
	// "待发货" = accepted（卖家已接受，未发货）
	// "待收货" = shipped（已发货）
	// "退款/售后" = rejected/cancelled
	// "待评价" = completed 且未评价
	var statusFilter string
	var extraJoin string
	switch tab {
	case "pending_pay":
		statusFilter = "AND status = 'pending'"
	case "pending_ship":
		statusFilter = "AND status = 'accepted'"
	case "pending_recv":
		statusFilter = "AND status = 'shipped'"
	case "refund":
		statusFilter = "AND status IN ('rejected','cancelled')"
	case "pending_review":
		// 已完成且当前角色未评价：buyer 查买家未评价，seller 查卖家未评价
		if role == "seller" {
			extraJoin = "LEFT JOIN reviews r ON orders.id = r.order_id AND r.reviewer_id = orders.seller_id"
		} else {
			extraJoin = "LEFT JOIN reviews r ON orders.id = r.order_id AND r.reviewer_id = orders.buyer_id"
		}
		statusFilter = "AND orders.status = 'completed' AND r.id IS NULL"
	default:
		statusFilter = ""
	}

	var err error
	var rows *sql.Rows

	// 注意：JOIN reviews 后所有列名都需用 orders. 前缀限定，避免歧义错误
	baseQuery := "SELECT orders.id, orders.product_id, orders.product_title, orders.product_image, orders.spec_name, orders.quantity, orders.buyer_id, orders.buyer_name, orders.seller_id, orders.seller_name, orders.price, orders.status, orders.message, COALESCE(orders.address_snapshot,''), orders.shipped_at, orders.created_at, orders.updated_at FROM orders"
	if extraJoin != "" {
		baseQuery += " " + extraJoin
	}

	ownerWhere := "WHERE orders.buyer_id = ?"
	if role != "buyer" {
		ownerWhere = "WHERE orders.seller_id = ?"
	}

	fullQuery := baseQuery + " " + ownerWhere + " " + statusFilter + " ORDER BY orders.created_at DESC"
	rows, err = db.Query(fullQuery, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	orders := []models.Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			continue
		}
		orders = append(orders, o)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "成功",
		Data:    models.PageData{List: orders, Total: len(orders), Page: 1, PageSize: 100},
	})
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	userID := c.GetString("userId")
	username := c.GetString("username")
	orderID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	validStatuses := map[string]bool{
		"accepted":  true,
		"rejected":  true,
		"shipped":   true,
		"completed": true,
		"cancelled": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的状态"})
		return
	}

	db := h.Store.GetDB()

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, COALESCE(address_snapshot,''), shipped_at, created_at, updated_at FROM orders WHERE id = ?", orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}

	if req.Status == "accepted" || req.Status == "rejected" || req.Status == "shipped" {
		if o.SellerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}
	if req.Status == "completed" {
		if o.BuyerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}
	if req.Status == "cancelled" {
		if o.BuyerID != userID && o.SellerID != userID {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权操作"})
			return
		}
	}

	// 状态机校验：防止非法状态流转（避免重复操作 / 倒退 / 跳级）
	// 合法流转:
	//   pending -> accepted | rejected | cancelled
	//   accepted -> shipped | cancelled
	//   shipped -> completed | cancelled
	//   completed/rejected/cancelled -> (终态，不允许再变更)
	legalTransitions := map[string]map[string]bool{
		"pending":  {"accepted": true, "rejected": true, "cancelled": true},
		"accepted": {"shipped": true, "cancelled": true},
		"shipped":  {"completed": true, "cancelled": true},
	}
	allowed, ok := legalTransitions[o.Status]
	if !ok || !allowed[req.Status] {
		statusMap := map[string]string{"pending": "待处理", "accepted": "已接受", "shipped": "已发货", "completed": "已完成", "rejected": "已拒绝", "cancelled": "已取消"}
		fromLabel := statusMap[o.Status]
		toLabel := statusMap[req.Status]
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "订单状态不允许从「" + fromLabel + "」变更为「" + toLabel + "」"})
		return
	}

	now := time.Now()

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}

	if req.Status == "shipped" {
		_, err = tx.Exec("UPDATE orders SET status = ?, shipped_at = ?, updated_at = ? WHERE id = ?", req.Status, now, now, orderID)
	} else {
		_, err = tx.Exec("UPDATE orders SET status = ?, updated_at = ? WHERE id = ?", req.Status, now, orderID)
	}
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新订单失败"})
		return
	}

	if req.Status == "rejected" || req.Status == "cancelled" {
		// 恢复规格库存 & 若商品是sold_out则改回selling
		if o.SpecName != "" {
			var specsJSON string
			if err := tx.QueryRow("SELECT specs FROM products WHERE id = ? FOR UPDATE", o.ProductID).Scan(&specsJSON); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询商品规格失败"})
				return
			}
			if specsJSON != "" {
				var specs []models.ProductSpec
				json.Unmarshal([]byte(specsJSON), &specs)
				for i := range specs {
					if specs[i].Name == o.SpecName || specs[i].ID == o.SpecName {
						if specs[i].Stock >= 0 {
							specs[i].Stock += o.Quantity
						}
						break
					}
				}
				newSpecsJSON, _ := json.Marshal(specs)
				if _, err := tx.Exec("UPDATE products SET specs = ? WHERE id = ?", string(newSpecsJSON), o.ProductID); err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品规格失败"})
					return
				}
			}
		}
		// 恢复商品为上架状态
		if _, err := tx.Exec("UPDATE products SET status = 'selling' WHERE id = ? AND status IN ('sold_out', 'sold')", o.ProductID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新商品状态失败"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}

	o.Status = req.Status
	o.UpdatedAt = now
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "操作成功", Data: o})

	// 发送系统消息通知对方
	if h.Chat != nil {
		// 确定谁对谁操作了啥
		action := req.Status // "accepted", "rejected", "shipped", "completed", "cancelled"
		switch action {
		case "accepted", "rejected", "shipped":
			// 卖家操作，通知买家
		case "completed":
			// 买家确认收货，通知卖家
		case "cancelled":
			// 任意一方取消，通知对方
		}
		h.Chat.SendSystemMsg(orderID, userID, username, action, o.ProductTitle, o.ProductImage, o.SpecName, o.Price, o.Quantity)
	}
}

func (h *OrderHandler) Get(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("userId")

	db := h.Store.GetDB()

	o, err := scanOrderRow(db.QueryRow("SELECT id, product_id, product_title, product_image, spec_name, quantity, buyer_id, buyer_name, seller_id, seller_name, price, status, message, COALESCE(address_snapshot,''), shipped_at, created_at, updated_at FROM orders WHERE id = ?", orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "订单不存在"})
		return
	}

	if o.BuyerID != userID && o.SellerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权查看"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "成功", Data: o})
}

func scanOrder(rows *sql.Rows) (models.Order, error) {
	var o models.Order
	var shippedAt sql.NullTime
	err := rows.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.ProductImage, &o.SpecName, &o.Quantity,
		&o.BuyerID, &o.BuyerName, &o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &o.AddressSnapshot, &shippedAt, &o.CreatedAt, &o.UpdatedAt)
	if shippedAt.Valid {
		o.ShippedAt = &shippedAt.Time
	}
	return o, err
}

func scanOrderRow(row *sql.Row) (models.Order, error) {
	var o models.Order
	var shippedAt sql.NullTime
	err := row.Scan(&o.ID, &o.ProductID, &o.ProductTitle, &o.ProductImage, &o.SpecName, &o.Quantity,
		&o.BuyerID, &o.BuyerName, &o.SellerID, &o.SellerName, &o.Price, &o.Status, &o.Message, &o.AddressSnapshot, &shippedAt, &o.CreatedAt, &o.UpdatedAt)
	if shippedAt.Valid {
		o.ShippedAt = &shippedAt.Time
	}
	return o, err
}
