package handlers

import (
	"encoding/json"
	"net/http"
	"school-trade/middleware"
	"school-trade/models"
	"school-trade/store"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type ChatHandler struct {
	Store *store.DBStore
	rooms map[string]map[*websocket.Conn]bool
	mu    sync.Mutex
}

func NewChatHandler(s *store.DBStore) *ChatHandler {
	return &ChatHandler{Store: s, rooms: make(map[string]map[*websocket.Conn]bool)}
}

func (h *ChatHandler) ChatWS(c *gin.Context) {
	// WebSocket 鉴权：从 query 参数读取 token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证"})
		return
	}
	claims, err := middleware.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证无效"})
		return
	}
	userID := claims.UserID
	username := claims.Username
	orderID := c.Param("orderId")

	// 验证用户属于此订单
	db := h.Store.GetDB()
	var buyerID, sellerID string
	err = db.QueryRow("SELECT buyer_id, seller_id FROM orders WHERE id = ?", orderID).Scan(&buyerID, &sellerID)
	if err != nil || (userID != buyerID && userID != sellerID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.mu.Lock()
	if h.rooms[orderID] == nil {
		h.rooms[orderID] = make(map[*websocket.Conn]bool)
	}
	h.rooms[orderID][conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.rooms[orderID], conn)
		h.mu.Unlock()
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		json.Unmarshal(msgBytes, &msg)

		if msg.Type == "recall" {
			var recallReq struct {
				MessageID string `json:"messageId"`
			}
			json.Unmarshal(msgBytes, &recallReq)
			result, _ := db.Exec("UPDATE chat_messages SET recalled = 1 WHERE id = ? AND sender_id = ? AND created_at > ?", recallReq.MessageID, userID, time.Now().Add(-3*time.Minute))
			affected, _ := result.RowsAffected()
			if affected > 0 {
				// 获取原消息时间
				var createdAt time.Time
				db.QueryRow("SELECT created_at FROM chat_messages WHERE id = ?", recallReq.MessageID).Scan(&createdAt)
				recallBroadcast, _ := json.Marshal(gin.H{
					"type":      "message",
					"id":        recallReq.MessageID,
					"content":   "[消息已撤回]",
					"recalled":  true,
					"createdAt": createdAt,
				})
				h.broadcast(orderID, conn, recallBroadcast)
			}
			continue
		}

		if msg.Type == "delete" {
			var delReq struct {
				MessageID string `json:"messageId"`
			}
			json.Unmarshal(msgBytes, &delReq)
			// 追加到 deleted_by
			var currentDeleted string
			db.QueryRow("SELECT deleted_by FROM chat_messages WHERE id = ?", delReq.MessageID).Scan(&currentDeleted)
			newDeleted := currentDeleted
			if newDeleted != "" {
				newDeleted += ","
			}
			newDeleted += userID
			db.Exec("UPDATE chat_messages SET deleted_by = ? WHERE id = ?", newDeleted, delReq.MessageID)
			// 只通知自己
			conn.WriteJSON(gin.H{"type": "deleted", "messageId": delReq.MessageID})
			continue
		}

		now := time.Now()
		id := genID("msg")
		msgType := msg.Type
		if msgType == "" {
			msgType = "text"
		}

		_, err = db.Exec("INSERT INTO chat_messages (id, order_id, sender_id, sender_name, content, type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, orderID, userID, username, msg.Content, msgType, now)
		if err != nil {
			continue
		}

		// 广播给房间内所有人
		broadcastMsg, _ := json.Marshal(gin.H{
			"type":       "message",
			"id":         id,
			"orderId":    orderID,
			"senderId":   userID,
			"senderName": username,
			"content":    msg.Content,
			"msgType":    msgType,
			"createdAt":  now,
		})
		h.broadcast(orderID, conn, broadcastMsg)
	}
}

func (h *ChatHandler) broadcast(orderID string, senderConn *websocket.Conn, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.rooms[orderID] {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// ChatHistory 获取聊天记录
func (h *ChatHandler) ChatHistory(c *gin.Context) {
	userID := c.GetString("userId")
	orderID := c.Param("orderId")

	db := h.Store.GetDB()
	var buyerID, sellerID string
	err := db.QueryRow("SELECT buyer_id, seller_id FROM orders WHERE id = ?", orderID).Scan(&buyerID, &sellerID)
	if err != nil || (userID != buyerID && userID != sellerID) {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权访问"})
		return
	}

	rows, err := db.Query("SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at FROM chat_messages WHERE order_id = ? ORDER BY created_at ASC", orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var msgs []gin.H
	for rows.Next() {
		var m models.ChatMessage
		rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.SenderName, &m.Content, &m.Type, &m.Recalled, &m.DeletedBy, &m.CreatedAt)

		// 检查是否被当前用户删除
		deleted := false
		if m.DeletedBy != "" {
			for _, d := range splitString(m.DeletedBy, ",") {
				if d == userID {
					deleted = true
					break
				}
			}
		}
		if deleted {
			continue
		}

		if m.Recalled {
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": "[消息已撤回]", "type": "text", "recalled": true, "createdAt": m.CreatedAt})
		} else {
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": m.Content, "type": m.Type, "recalled": false, "createdAt": m.CreatedAt})
		}
	}
	if msgs == nil {
		msgs = []gin.H{}
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: msgs})
}

func splitString(s, sep string) []string {
	var result []string
	for _, part := range splitStr(s, sep) {
		p := trimStr(part)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// Simple split (standalone to avoid import)
func splitStr(s, sep string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimStr(s string) string {
	l, r := 0, len(s)-1
	for l <= r && (s[l] == ' ' || s[l] == '\t' || s[l] == '\n' || s[l] == '\r') {
		l++
	}
	for r >= l && (s[r] == ' ' || s[r] == '\t' || s[r] == '\n' || s[r] == '\r') {
		r--
	}
	return s[l : r+1]
}
