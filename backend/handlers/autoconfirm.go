package handlers

import (
	"database/sql"
	"log"
	"time"

	"school-trade/store"
)

// 自动确认收货定时任务
// 规则参考淘宝：实物商品发货后 7 天自动确认收货
// 本系统统一为 shipped 状态超过 7 天 → 自动转为 completed

const autoConfirmDuration = 7 * 24 * time.Hour

// StartAutoConfirm 启动自动确认收货的定时任务
// 每 1 小时扫描一次，将 shipped_at 早于 7 天前且 status='shipped' 的订单转为 completed
func StartAutoConfirm(s *store.DBStore) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		// 启动时立即执行一次
		runAutoConfirm(s)
		for range ticker.C {
			runAutoConfirm(s)
		}
	}()
	log.Println("[autoconfirm] 自动确认收货任务已启动，每 1 小时执行一次")
}

func runAutoConfirm(s *store.DBStore) {
	db := s.GetDB()
	if db == nil {
		return
	}
	threshold := time.Now().Add(-autoConfirmDuration)

	// 查询需要自动确认的订单
	rows, err := db.Query(`SELECT id FROM orders WHERE status = 'shipped' AND shipped_at IS NOT NULL AND shipped_at < ?`, threshold)
	if err != nil {
		log.Printf("[autoconfirm] 查询失败: %v", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		return
	}

	// 事务批量更新
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[autoconfirm] 事务启动失败: %v", err)
		return
	}
	now := time.Now()
	var committed []string
	for _, id := range ids {
		if _, err := tx.Exec("UPDATE orders SET status = 'completed', updated_at = ? WHERE id = ? AND status = 'shipped'", now, id); err != nil {
			log.Printf("[autoconfirm] 更新订单 %s 失败: %v", id, err)
			continue
		}
		committed = append(committed, id)
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[autoconfirm] 事务提交失败: %v", err)
		_ = tx.Rollback()
		return
	}
	log.Printf("[autoconfirm] 已自动确认收货 %d 单: %v", len(committed), committed)
}

// 兼容：防止 sql 包未使用
var _ = sql.NullTime{}
