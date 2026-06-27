package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
			result, err := db.Exec("UPDATE chat_messages SET recalled = 1 WHERE id = ? AND sender_id = ? AND created_at > ?", recallReq.MessageID, userID, time.Now().Add(-3*time.Minute))
			if err != nil {
				continue
			}
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
	// 快照当前房间内所有连接，释放锁后再写，避免慢客户端阻塞整个广播
	// 注：不跳过 senderConn — 前端 chat.html 依赖服务端回显来渲染自己发送的消息
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.rooms[peerKey]))
	for conn := range h.rooms[peerKey] {
		conns = append(conns, conn)
	}
	h.mu.Unlock()

	for _, conn := range conns {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// ChatHistory 获取聊天记录（按 peer_key 查询，同一个用户对共用聊天记录，分页20条）
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

	limit := 20
	before := c.Query("before")

	var rows *sql.Rows
	if before != "" {
		rows, err = db.Query(`SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at 
			FROM chat_messages WHERE peer_key = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`, peerKey, before, limit)
	} else {
		rows, err = db.Query(`SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at 
			FROM chat_messages WHERE peer_key = ? ORDER BY created_at DESC LIMIT ?`, peerKey, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var msgs []gin.H
	for rows.Next() {
		var m models.ChatMessage
		rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.SenderName, &m.Content, &m.Type, &m.Recalled, &m.DeletedBy, &m.CreatedAt)

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
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": "[消息已撤回]", "type": "text", "msgType": "text", "recalled": true, "createdAt": m.CreatedAt})
		} else {
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": m.Content, "type": m.Type, "msgType": m.Type, "recalled": false, "createdAt": m.CreatedAt})
		}
	}
	if msgs == nil {
		msgs = []gin.H{}
	}

	hasMore := len(msgs) >= limit

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"messages": msgs, "hasMore": hasMore}})
}

// ChatList 获取当前用户的所有对话列表（类似QQ）
func (h *ChatHandler) ChatList(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	// 获取用户参与的所有peer_key及最后一条消息
	// peer_key 格式为 "userA:userB"（按字母序排序），用 SUBSTRING_INDEX 精确匹配两端，
	// 避免使用 LIKE '%userID%' 误匹配包含用户ID子串的其他对话
	rows, err := db.Query(`
		SELECT cm.peer_key, cm.content, cm.type, cm.sender_id, cm.sender_name, cm.created_at,
		       CASE WHEN cm.sender_id = ? THEN 0 ELSE 1 END AS is_other
		FROM chat_messages cm
		INNER JOIN (
			SELECT peer_key, MAX(created_at) AS last_time
			FROM chat_messages
			WHERE SUBSTRING_INDEX(peer_key, ':', 1) = ?
			   OR SUBSTRING_INDEX(peer_key, ':', -1) = ?
			GROUP BY peer_key
		) latest ON cm.peer_key = latest.peer_key AND cm.created_at = latest.last_time
		ORDER BY cm.created_at DESC
	`, userID, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	type ConvInfo struct {
		PeerKey     string    `json:"peerKey"`
		PeerName    string    `json:"peerName"`
		PeerAvatar  string    `json:"peerAvatar"`
		PeerID      string    `json:"peerId"`
		LastMsg     string    `json:"lastMsg"`
		LastMsgType string    `json:"lastMsgType"`
		LastTime    time.Time `json:"lastTime"`
		UnreadCount int       `json:"unreadCount"`
	}

	var list []ConvInfo
	for rows.Next() {
		var info ConvInfo
		var senderID, senderName string
		var msgContent, msgType string
		var dummyIsOther int
		rows.Scan(&info.PeerKey, &msgContent, &msgType, &senderID, &senderName, &info.LastTime, &dummyIsOther)
		if msgType == "image" {
			info.LastMsg = "[图片]"
		} else if msgType == "system" {
			var sysContent map[string]interface{}
			if json.Unmarshal([]byte(msgContent), &sysContent) == nil {
				actionMap := map[string]string{
					"bought": "购买了", "accepted": "接受了订单", "rejected": "拒绝了订单",
					"shipped": "已发货", "completed": "已确认收货", "cancelled": "取消了订单",
				}
				name, _ := sysContent["text"].(string)
				act, _ := sysContent["action"].(string)
				title, _ := sysContent["productTitle"].(string)
				if verb, ok := actionMap[act]; ok {
					info.LastMsg = name + " " + verb + " 「" + title + "」"
				} else {
					info.LastMsg = "[系统通知]"
				}
			} else {
				info.LastMsg = "[系统通知]"
			}
		} else {
			info.LastMsg = truncateStr(msgContent, 50)
		}
		info.LastMsgType = msgType

		// 解析对方信息
		parts := strings.SplitN(info.PeerKey, ":", 2)
		if len(parts) == 2 {
			if parts[0] == userID {
				info.PeerID = parts[1]
			} else {
				info.PeerID = parts[0]
			}
		}
		// 获取对方昵称与头像
		db.QueryRow("SELECT nickname, avatar FROM users WHERE id = ?", info.PeerID).Scan(&info.PeerName, &info.PeerAvatar)
		if info.PeerName == "" {
			info.PeerName = info.PeerID
		}

		// 查询未读数
		var lastRead sql.NullTime
		db.QueryRow("SELECT last_read_at FROM chat_read_cursors WHERE user_id = ? AND peer_key = ?", userID, info.PeerKey).Scan(&lastRead)
		if lastRead.Valid {
			db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE peer_key = ? AND sender_id != ? AND created_at > ?",
				info.PeerKey, userID, lastRead.Time).Scan(&info.UnreadCount)
		} else {
			db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE peer_key = ? AND sender_id != ?",
				info.PeerKey, userID).Scan(&info.UnreadCount)
		}

		list = append(list, info)
	}
	if list == nil {
		list = []ConvInfo{}
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: list})
}

// ChatUnreadCount 获取未读消息总数
func (h *ChatHandler) ChatUnreadCount(c *gin.Context) {
	userID := c.GetString("userId")
	db := h.Store.GetDB()

	total := 0
	// 获取用户参与的所有peer_key（精确匹配两端，避免子串误匹配）
	rows, err := db.Query(`
		SELECT DISTINCT peer_key FROM chat_messages
		WHERE SUBSTRING_INDEX(peer_key, ':', 1) = ?
		   OR SUBSTRING_INDEX(peer_key, ':', -1) = ?
	`, userID, userID)
	if err != nil {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"count": 0}})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var peerKey string
		rows.Scan(&peerKey)
		var lastRead sql.NullTime
		db.QueryRow("SELECT last_read_at FROM chat_read_cursors WHERE user_id = ? AND peer_key = ?", userID, peerKey).Scan(&lastRead)
		if lastRead.Valid {
			var cnt int
			db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE peer_key = ? AND sender_id != ? AND created_at > ?",
				peerKey, userID, lastRead.Time).Scan(&cnt)
			total += cnt
		} else {
			var cnt int
			db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE peer_key = ? AND sender_id != ?",
				peerKey, userID).Scan(&cnt)
			total += cnt
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"count": total}})
}

// ChatRead 标记某个对话为已读
func (h *ChatHandler) ChatRead(c *gin.Context) {
	userID := c.GetString("userId")
	peerKey := c.Query("peer_key")
	if peerKey == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "缺少 peer_key 参数"})
		return
	}

	db := h.Store.GetDB()
	now := time.Now()
	_, err := db.Exec(`INSERT INTO chat_read_cursors (user_id, peer_key, last_read_at) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE last_read_at = ?`, userID, peerKey, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "标记失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "ok"})
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

// ========== 基于 peer_key 的聊天（不依赖订单） ==========

// ChatHistoryPeer 通过 peer_key 获取聊天记录（分页，每页20条）
func (h *ChatHandler) ChatHistoryPeer(c *gin.Context) {
	userID := c.GetString("userId")
	peerKey := c.Query("peer_key")
	if peerKey == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "缺少 peer_key 参数"})
		return
	}
	// 验证用户是否属于该 peer
	parts := strings.SplitN(peerKey, ":", 2)
	if len(parts) != 2 || (userID != parts[0] && userID != parts[1]) {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权访问"})
		return
	}

	limit := 20
	before := c.Query("before") // createdAt 时间戳，加载此时间之前的消息

	db := h.Store.GetDB()
	var rows *sql.Rows
	var err error
	if before != "" {
		rows, err = db.Query(`SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at 
			FROM chat_messages WHERE peer_key = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`, peerKey, before, limit)
	} else {
		rows, err = db.Query(`SELECT id, order_id, sender_id, sender_name, content, type, recalled, deleted_by, created_at 
			FROM chat_messages WHERE peer_key = ? ORDER BY created_at DESC LIMIT ?`, peerKey, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var msgs []gin.H
	rawCount := 0
	for rows.Next() {
		rawCount++
		var m models.ChatMessage
		rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.SenderName, &m.Content, &m.Type, &m.Recalled, &m.DeletedBy, &m.CreatedAt)

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
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": "[消息已撤回]", "type": "text", "msgType": "text", "recalled": true, "createdAt": m.CreatedAt})
		} else {
			msgs = append(msgs, gin.H{"id": m.ID, "orderId": m.OrderID, "senderId": m.SenderID, "senderName": m.SenderName, "content": m.Content, "type": m.Type, "msgType": m.Type, "recalled": false, "createdAt": m.CreatedAt})
		}
	}
	if msgs == nil {
		msgs = []gin.H{}
	}

	hasMore := rawCount >= limit

	// 获取双方头像，供前端在聊天头部和消息气泡旁显示
	peerID := ""
	if parts := strings.SplitN(peerKey, ":", 2); len(parts) == 2 {
		if parts[0] == userID {
			peerID = parts[1]
		} else {
			peerID = parts[0]
		}
	}
	var peerAvatar, myAvatar string
	db.QueryRow("SELECT COALESCE(avatar,'') FROM users WHERE id = ?", peerID).Scan(&peerAvatar)
	db.QueryRow("SELECT COALESCE(avatar,'') FROM users WHERE id = ?", userID).Scan(&myAvatar)

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"messages": msgs, "hasMore": hasMore, "peerAvatar": peerAvatar, "peerId": peerID, "myAvatar": myAvatar}})
}

// ChatWSPeer WebSocket 连接（通过 peer_key，不依赖订单）
func (h *ChatHandler) ChatWSPeer(c *gin.Context) {
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
	peerKey := c.Query("peer_key")
	if peerKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 peer_key 参数"})
		return
	}

	// 验证 peer_key 格式和用户归属
	parts := strings.SplitN(peerKey, ":", 2)
	if len(parts) != 2 || (userID != parts[0] && userID != parts[1]) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}

	db := h.Store.GetDB()

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
			result, err := db.Exec("UPDATE chat_messages SET recalled = 1 WHERE id = ? AND sender_id = ? AND created_at > ?", recallReq.MessageID, userID, time.Now().Add(-3*time.Minute))
			if err != nil {
				continue
			}
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

		// order_id 可以为空（无订单聊天）
		_, err = db.Exec("INSERT INTO chat_messages (id, order_id, peer_key, sender_id, sender_name, content, type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			id, "", peerKey, userID, username, msg.Content, msgType, now)
		if err != nil {
			continue
		}

		broadcastMsg, _ := json.Marshal(gin.H{
			"type":       "message",
			"id":         id,
			"orderId":    "",
			"senderId":   userID,
			"senderName": username,
			"content":    msg.Content,
			"msgType":    msgType,
			"createdAt":  now,
		})
		h.broadcast(peerKey, conn, broadcastMsg)
	}
}

// UploadChatImage 上传聊天图片，保存到 resources/chat/ 目录
func (h *ChatHandler) UploadChatImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请选择图片文件"})
		return
	}
	defer file.Close()

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "图片大小不能超过 10MB"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不支持的图片格式，仅支持 jpg/png/gif/webp/bmp"})
		return
	}

	execDir, _ := os.Getwd()
	uploadDir := filepath.Join(execDir, "..", "frontend", "resources", "chat")
	if _, err := os.Stat(filepath.Join(execDir, "frontend")); err == nil {
		uploadDir = filepath.Join(execDir, "frontend", "resources", "chat")
	}
	os.MkdirAll(uploadDir, 0755)

	fileName := fmt.Sprintf("chat_%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "保存图片失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "写入图片失败"})
		return
	}

	imageURL := "/resources/chat/" + fileName
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "上传成功", Data: gin.H{"url": imageURL}})
}

// InitChat 发起聊天 — 创建或获取与卖家的 peer 对话（无需订单）
func (h *ChatHandler) InitChat(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		PeerID   string `json:"peerId"`
		PeerName string `json:"peerName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PeerID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if userID == req.PeerID {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不能和自己聊天"})
		return
	}

	peerKey := makePeerKey(userID, req.PeerID)

	// 尝试获取对方昵称
	db := h.Store.GetDB()
	var peerName string
	db.QueryRow("SELECT nickname FROM users WHERE id = ?", req.PeerID).Scan(&peerName)
	if peerName == "" {
		peerName = req.PeerName
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{
		"peerKey":  peerKey,
		"peerName": peerName,
		"peerId":   req.PeerID,
	}})
}

// SendSystemMsg 发送系统消息（订单状态变更通知），通过WebSocket实时推送给对方
func (h *ChatHandler) SendSystemMsg(orderID, fromUserID, fromUserName, action, productTitle, productImage, specName string, price float64, quantity int) {
	db := h.Store.GetDB()

	// 解析订单获取双方信息
	var buyerID, sellerID string
	if err := db.QueryRow("SELECT buyer_id, seller_id FROM orders WHERE id = ?", orderID).Scan(&buyerID, &sellerID); err != nil {
		return
	}

	peerKey := makePeerKey(buyerID, sellerID)

	// 确定目标用户
	targetID := sellerID
	if fromUserID == sellerID {
		targetID = buyerID
	}

	// 系统消息发送者视为一个虚拟的"系统"账号（使用system前缀避免和真实用户冲突）
	sysSenderID := "system"
	sysSenderName := "系统通知"

	content := map[string]interface{}{
		"action":       action,
		"text":         fromUserName,
		"productTitle": productTitle,
		"productImage": productImage,
		"specName":     specName,
		"orderId":      orderID,
		"price":        price,
		"quantity":     quantity,
	}
	contentJSON, _ := json.Marshal(content)

	now := time.Now()
	id := genID("sys")

	_, err := db.Exec(
		"INSERT INTO chat_messages (id, order_id, peer_key, sender_id, sender_name, content, type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, orderID, peerKey, sysSenderID, sysSenderName, string(contentJSON), "system", now,
	)
	if err != nil {
		return
	}

	// WebSocket 广播
	broadcastMsg, _ := json.Marshal(gin.H{
		"type":       "message",
		"id":         id,
		"orderId":    orderID,
		"senderId":   sysSenderID,
		"senderName": sysSenderName,
		"content":    string(contentJSON),
		"msgType":    "system",
		"targetId":   targetID,
		"createdAt":  now,
	})
	h.broadcast(peerKey, nil, broadcastMsg)
}
