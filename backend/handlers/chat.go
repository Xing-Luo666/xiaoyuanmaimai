package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"school-trade/middleware"
	"school-trade/models"
	"school-trade/store"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type ChatHandler struct {
	Store *store.DBStore
	rooms map[string]map[*websocket.Conn]bool // key=peer_key
	mu    sync.Mutex
}

func NewChatHandler(s *store.DBStore) *ChatHandler {
	return &ChatHandler{Store: s, rooms: make(map[string]map[*websocket.Conn]bool)}
}

// makePeerKey 生成用户对的唯一标识（按字母序排序）
func makePeerKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// resolvePeer 从订单 ID 解析出用户对，返回 peer_key
func (h *ChatHandler) resolvePeer(orderID string) string {
	db := h.Store.GetDB()
	var buyerID, sellerID string
	err := db.QueryRow("SELECT buyer_id, seller_id FROM orders WHERE id = ?", orderID).Scan(&buyerID, &sellerID)
	if err != nil {
		return ""
	}
	return makePeerKey(buyerID, sellerID)
}

func (h *ChatHandler) ChatWS(c *gin.Context) {
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

	db := h.Store.GetDB()
	var buyerID, sellerID string
	err = db.QueryRow("SELECT buyer_id, seller_id FROM orders WHERE id = ?", orderID).Scan(&buyerID, &sellerID)
	if err != nil || (userID != buyerID && userID != sellerID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}

	// 使用 peer_key 作为房间标识
	peerKey := makePeerKey(buyerID, sellerID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.mu.Lock()
	if h.rooms[peerKey] == nil {
		h.rooms[peerKey] = make(map[*websocket.Conn]bool)
	}
	h.rooms[peerKey][conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.rooms[peerKey], conn)
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
				var createdAt time.Time
				db.QueryRow("SELECT created_at FROM chat_messages WHERE id = ?", recallReq.MessageID).Scan(&createdAt)
				recallBroadcast, _ := json.Marshal(gin.H{
					"type":      "message",
					"id":        recallReq.MessageID,
					"content":   "[消息已撤回]",
					"recalled":  true,
					"createdAt": createdAt,
				})
				h.broadcast(peerKey, conn, recallBroadcast)
			}
			continue
		}

		if msg.Type == "delete" {
			var delReq struct {
				MessageID string `json:"messageId"`
			}
			json.Unmarshal(msgBytes, &delReq)
			var currentDeleted string
			db.QueryRow("SELECT deleted_by FROM chat_messages WHERE id = ?", delReq.MessageID).Scan(&currentDeleted)
			newDeleted := currentDeleted
			if newDeleted != "" {
				newDeleted += ","
			}
			newDeleted += userID
			db.Exec("UPDATE chat_messages SET deleted_by = ? WHERE id = ?", newDeleted, delReq.MessageID)
			conn.WriteJSON(gin.H{"type": "deleted", "messageId": delReq.MessageID})
			continue
		}

		now := time.Now()
		id := genID("msg")
		msgType := msg.Type
		if msgType == "" {
			msgType = "text"
		}

		_, err = db.Exec("INSERT INTO chat_messages (id, order_id, peer_key, sender_id, sender_name, content, type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			id, orderID, peerKey, userID, username, msg.Content, msgType, now)
		if err != nil {
			continue
		}

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
		h.broadcast(peerKey, conn, broadcastMsg)
	}
}

func (h *ChatHandler) broadcast(peerKey string, senderConn *websocket.Conn, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.rooms[peerKey] {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// ChatHistory 获取聊天记录（按 peer_key 查询，同一个用户对共用聊天记录）
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

	peerKey := makePeerKey(buyerID, sellerID)

	rows, err := db.Query("SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at FROM chat_messages WHERE peer_key = ? ORDER BY created_at ASC", peerKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var msgs []gin.H
	for rows.Next() {
		var m models.ChatMessage
		var dummyPeer string
		rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.SenderName, &m.Content, &m.Type, &m.Recalled, &m.DeletedBy, &m.CreatedAt)
		_ = dummyPeer

		deleted := false
		if m.DeletedBy != "" {
			for _, d := range splitStr(m.DeletedBy, ",") {
				if strings.TrimSpace(d) == userID {
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

// ========== 管理员接口 ==========

// AdminChatPeers 列出所有聊天用户对
func (h *ChatHandler) AdminChatPeers(c *gin.Context) {
	db := h.Store.GetDB()
	rows, err := db.Query("SELECT DISTINCT peer_key, COUNT(*) AS msg_cnt, MAX(created_at) AS last_msg FROM chat_messages WHERE peer_key != '' GROUP BY peer_key ORDER BY last_msg DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	type PeerInfo struct {
		PeerKey  string    `json:"peerKey"`
		UserA    string    `json:"userA"`
		UserB    string    `json:"userB"`
		MsgCount int       `json:"msgCount"`
		LastMsg  time.Time `json:"lastMsg"`
	}

	var list []PeerInfo
	for rows.Next() {
		var p PeerInfo
		var last sql.NullTime
		rows.Scan(&p.PeerKey, &p.MsgCount, &last)
		if last.Valid {
			p.LastMsg = last.Time
		}
		parts := strings.SplitN(p.PeerKey, ":", 2)
		if len(parts) == 2 {
			p.UserA = parts[0]
			p.UserB = parts[1]
		}
		list = append(list, p)
	}
	if list == nil {
		list = []PeerInfo{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

// AdminChatMessages 查看指定用户对的所有聊天记录
func (h *ChatHandler) AdminChatMessages(c *gin.Context) {
	peerKey := c.Query("peer_key")
	if peerKey == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "缺少 peer_key 参数"})
		return
	}

	db := h.Store.GetDB()
	rows, err := db.Query("SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at FROM chat_messages WHERE peer_key = ? ORDER BY created_at ASC", peerKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var msgs []gin.H
	for rows.Next() {
		var m models.ChatMessage
		rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.SenderName, &m.Content, &m.Type, &m.Recalled, &m.DeletedBy, &m.CreatedAt)
		msgs = append(msgs, gin.H{
			"id":         m.ID,
			"orderId":    m.OrderID,
			"senderId":   m.SenderID,
			"senderName": m.SenderName,
			"content":    truncateStr(m.Content, 200),
			"type":       m.Type,
			"recalled":   m.Recalled,
			"createdAt":  m.CreatedAt,
		})
	}
	if msgs == nil {
		msgs = []gin.H{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: msgs})
}

// AdminChatDelete 删除指定用户对的所有聊天记录
func (h *ChatHandler) AdminChatDelete(c *gin.Context) {
	peerKey := c.Query("peer_key")
	if peerKey == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "缺少 peer_key 参数"})
		return
	}

	db := h.Store.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "开启事务失败"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	result, err := tx.Exec("DELETE FROM chat_messages WHERE peer_key = ?", peerKey)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "提交事务失败: " + err.Error()})
		return
	}

	affected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: fmt.Sprintf("已删除 %d 条聊天记录", affected)})
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func splitStr(s, sep string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	for {
		idx := strings.Index(s, sep)
		if idx < 0 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+len(sep):]
	}
	return parts
}
