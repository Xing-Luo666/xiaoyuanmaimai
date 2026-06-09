package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"school-trade/store"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var allowedTables = map[string]bool{
	"users":         true,
	"products":      true,
	"orders":        true,
	"cart_items":    true,
	"favorites":     true,
	"history":       true,
	"chat_messages": true,
}

type AdminHandler struct {
	Store *store.DBStore
}

func NewAdminHandler(s *store.DBStore) *AdminHandler {
	return &AdminHandler{Store: s}
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Key      string `json:"key"`
	Default  string `json:"default"`
	Extra    string `json:"extra"`
}

type TableInfo struct {
	Name     string       `json:"name"`
	Columns  []ColumnInfo `json:"columns"`
	RowCount int          `json:"rowCount"`
}

func (h *AdminHandler) ListTables(c *gin.Context) {
	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	rows, err := db.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询表列表失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if !allowedTables[name] {
			continue
		}

		cols, _ := h.getTableColumns(db, name)
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", name)).Scan(&count)

		tables = append(tables, TableInfo{
			Name:     name,
			Columns:  cols,
			RowCount: count,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": tables})
}

func (h *AdminHandler) getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	rows, err := db.Query(
		"SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, IFNULL(COLUMN_DEFAULT,''), EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		rows.Scan(&c.Name, &c.Type, &nullable, &c.Key, &c.Default, &c.Extra)
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, nil
}

func (h *AdminHandler) ListRows(c *gin.Context) {
	tableName := c.Param("table")
	if !allowedTables[tableName] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不允许的表名"})
		return
	}

	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	cols, err := h.getTableColumns(db, tableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取列信息失败: " + err.Error()})
		return
	}

	query := fmt.Sprintf("SELECT * FROM `%s` ORDER BY 1 DESC LIMIT 500", tableName)
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询数据失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var data []map[string]interface{}
	colNames, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		row := make(map[string]interface{})
		for i, col := range colNames {
			val := values[i]
			if val == nil {
				row[col] = nil
				continue
			}
			switch v := val.(type) {
			case []byte:
				// 尝试解析 JSON 字段
				s := string(v)
				if (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) || (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
					var js interface{}
					if json.Unmarshal(v, &js) == nil {
						row[col] = js
						continue
					}
				}
				row[col] = s
			default:
				row[col] = v
			}
		}
		data = append(data, row)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"columns": cols, "rows": data}})
}

func (h *AdminHandler) GetRow(c *gin.Context) {
	tableName := c.Param("table")
	rowID := c.Param("id")
	if !allowedTables[tableName] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不允许的表名"})
		return
	}

	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE id = ?", tableName)
	row := db.QueryRow(query, rowID)

	cols, _ := h.getTableColumns(db, tableName)
	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.Name
	}

	values := make([]interface{}, len(colNames))
	valuePtrs := make([]interface{}, len(colNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}

	data := make(map[string]interface{})
	for i, col := range colNames {
		val := values[i]
		if val == nil {
			data[col] = nil
			continue
		}
		switch v := val.(type) {
		case []byte:
			data[col] = string(v)
		default:
			data[col] = v
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data})
}

func (h *AdminHandler) CreateRow(c *gin.Context) {
	tableName := c.Param("table")
	if !allowedTables[tableName] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不允许的表名"})
		return
	}

	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 使用事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "开启事务失败: " + err.Error()})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cols, err := h.getTableColumns(db, tableName)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取列信息失败: " + err.Error()})
		return
	}

	// 检查必填字段（NOT NULL 且无 DEFAULT 且非 AUTO_INCREMENT）
	requiredCols := make(map[string]bool)
	for _, col := range cols {
		if !col.Nullable && col.Default == "" && col.Extra != "auto_increment" {
			requiredCols[col.Name] = true
		}
	}

	var placeholders []string
	var values []interface{}
	var colList []string

	for _, col := range cols {
		if col.Extra == "auto_increment" {
			continue
		}
		val, exists := req[col.Name]
		if !exists {
			// id 字段自动生成
			if col.Name == "id" && requiredCols[col.Name] {
				idPrefix := tableName
				if idPrefix == "users" {
					idPrefix = "u"
				} else if idPrefix == "products" {
					idPrefix = "p"
				} else if idPrefix == "orders" {
					idPrefix = "o"
				}
				b := make([]byte, 6)
				if _, err := rand.Read(b); err != nil {
					b = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
				}
				val = idPrefix + "-" + hex.EncodeToString(b)
				req[col.Name] = val
			} else if requiredCols[col.Name] {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少必填字段: " + col.Name})
				return
			} else {
				continue
			}
		}
		colList = append(colList, "`"+col.Name+"`")
		placeholders = append(placeholders, "?")

		// 处理 JSON 类型字段
		if strings.Contains(strings.ToLower(col.Type), "json") {
			jsonBytes, err := json.Marshal(val)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "字段 " + col.Name + " JSON 格式错误"})
				return
			}
			values = append(values, string(jsonBytes))
		} else {
			values = append(values, val)
		}
	}

	if len(colList) == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有可插入的字段"})
		return
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", tableName, strings.Join(colList, ", "), strings.Join(placeholders, ", "))
	result, err := tx.Exec(query, values...)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "插入失败: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交事务失败: " + err.Error()})
		return
	}

	lastID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "创建成功", "data": gin.H{"lastInsertId": lastID, "rowsAffected": rowsAffected}})
}

func (h *AdminHandler) UpdateRow(c *gin.Context) {
	tableName := c.Param("table")
	rowID := c.Param("id")
	if !allowedTables[tableName] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不允许的表名"})
		return
	}

	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 使用事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "开启事务失败: " + err.Error()})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 先检查记录是否存在
	var exists int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE id = ?", tableName), rowID).Scan(&exists)
	if err != nil || exists == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}

	cols, err := h.getTableColumns(db, tableName)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取列信息失败: " + err.Error()})
		return
	}

	var setClauses []string
	var values []interface{}

	for _, col := range cols {
		// 不允许修改 id 和 auto_increment 列
		if col.Name == "id" || col.Extra == "auto_increment" {
			continue
		}
		val, exists := req[col.Name]
		if !exists {
			continue
		}
		setClauses = append(setClauses, "`"+col.Name+"` = ?")

		if strings.Contains(strings.ToLower(col.Type), "json") {
			jsonBytes, err := json.Marshal(val)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "字段 " + col.Name + " JSON 格式错误"})
				return
			}
			values = append(values, string(jsonBytes))
		} else {
			values = append(values, val)
		}
	}

	if len(setClauses) == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有需要更新的字段"})
		return
	}

	values = append(values, rowID)
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?", tableName, strings.Join(setClauses, ", "))
	result, err := tx.Exec(query, values...)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交事务失败: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功", "data": gin.H{"rowsAffected": rowsAffected}})
}

func (h *AdminHandler) DeleteRow(c *gin.Context) {
	tableName := c.Param("table")
	rowID := c.Param("id")
	if !allowedTables[tableName] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不允许的表名"})
		return
	}

	db := h.Store.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "数据库未连接"})
		return
	}

	// 使用事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "开启事务失败: " + err.Error()})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 先检查记录是否存在
	var exists int
	err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE id = ?", tableName), rowID).Scan(&exists)
	if err != nil || exists == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}

	result, err := tx.Exec(fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", tableName), rowID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交事务失败: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功", "data": gin.H{"rowsAffected": rowsAffected}})
}
