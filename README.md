# 校园二手交易系统 - Docker 部署文档

## 项目概述

本项目是一个校园二手交易平台，包含三个主要服务：

| 服务 | 端口 | 说明 |
|------|------|------|
| **app** | 28080 | 校园二手交易主后端（Go + Gin） |
| **uas-app** | 8081 | UAS 统一身份认证后端（Go + Gin） |
| **uas-frontend** | 8082 | UAS 前端管理界面（Vue.js SPA + Nginx） |
| **mysql** | 3306 | MySQL 8.0 数据库 |

## 前提条件

- Docker Engine >= 20.10
- Docker Compose >= 1.29
- 服务器已开放对应端口（28080, 8081, 8082, 3306）

## 项目结构

```
.
├── docker-compose.yml        # Docker Compose 编排文件
├── Dockerfile                 # 主后端 app 构建文件
├── Dockerfile.uas             # UAS 后端构建文件
├── .dockerignore              # Docker 构建忽略规则
├── backend/                   # 主后端源码（Go）
│   ├── main.go
│   ├── go.mod
│   └── handlers/
├── frontend/                  # 主前端资源
├── uas/
│   ├── backend/               # UAS 后端源码（Go）
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── config/
│   │   └── ...
│   ├── frontend/
│   │   └── dist/              # UAS 前端构建产物（Vue.js SPA）
│   ├── docs/
│   │   └── init_tables.sql    # MySQL 初始化 SQL（自动建表）
│   └── nginx.conf             # Nginx 反向代理配置
```

## 快速部署

### 1. 克隆代码

```bash
git clone https://github.com/Xing-Luo666/xiaoyuanmaimai.git
cd xiaoyuanmaimai
```

### 2. 启动所有服务

```bash
docker-compose up -d
```

首次启动会自动构建镜像，耗时约 1-2 分钟。

### 3. 查看服务状态

```bash
docker-compose ps
```

所有服务状态应为 `Up`。

### 4. 验证部署

```bash
# 验证主后端
curl http://localhost:28080/health

# 验证 UAS 后端
curl http://localhost:8081/api/health

# 访问 UAS 前端
# 浏览器打开 http://<服务器IP>:8082
```

## 服务详情

### MySQL 数据库

- 镜像：`mysql:8.0`
- 端口：`3306`
- 数据库：`school_trade`（主业务）、`uas_db`（认证）
- 数据持久化：`mysql_data` 卷
- 健康检查：MySQL ping
- 初始化脚本：`uas/docs/init_tables.sql` 自动挂载到容器启动目录

启动时会自动创建 UAS 相关表（`u_user`, `u_corp_user`, `u_app`, `u_login_log`, `sys_user`, `sys_role`, `sys_menu` 等）。

### 主后端（app）

- 端口：`28080`（可通过环境变量 `PORT` 覆盖）
- 构建方式：Go 多阶段构建（编译 → Alpine 运行）
- 健康检查：`GET /health`
- 依赖：等待 MySQL 健康后才启动

### UAS 后端（uas-app）

- 端口：`8081`（可通过环境变量 `UAS_PORT` 覆盖）
- 构建方式：Go 多阶段构建
- 健康检查：`GET /api/health`
- 环境变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `UAS_PORT` | 8081 | 监听端口 |
| `DB_HOST` | mysql | 数据库主机 |
| `DB_PORT` | 3306 | 数据库端口 |
| `DB_USER` | root | 数据库用户 |
| `DB_PASSWORD` | 114514 | 数据库密码 |
| `DB_NAME` | uas_db | 数据库名 |
| `JWT_SECRET` | uas-secret-key-2026-school-trade | JWT 密钥 |
| `JWT_EXPIRE_HOURS` | 24 | JWT 过期时间 |

### UAS 前端（uas-frontend）

- 镜像：`nginx:alpine`
- 端口：`8082`（可通过环境变量 `UAS_FE_PORT` 覆盖）
- 静态文件：`uas/frontend/dist/` → Nginx `/usr/share/nginx/html`
- 前端路由：支持 Vue.js history 模式（`try_files` 回退）
- API 代理：`/api/` → `http://uas-app:8081`（反向代理）

## Dockerfile 构建说明

### Dockerfile（主后端）

```dockerfile
# 阶段 1：编译
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o school-trade .

# 阶段 2：运行
FROM alpine:3.20
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/school-trade .
COPY frontend/ /frontend
EXPOSE 28080
CMD ["./school-trade"]
```

### Dockerfile.uas（UAS 后端）

```dockerfile
# 阶段 1：编译
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY uas/backend/go.mod uas/backend/go.sum ./
RUN go mod download
COPY uas/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o uas .

# 阶段 2：运行
FROM alpine:3.20
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/uas .
RUN mkdir -p /app/uploads/avatar
EXPOSE 8081
CMD ["./uas"]
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_PASSWORD` | 114514 | MySQL root 密码 |
| `DB_NAME` | school_trade | 主业务数据库名 |
| `PORT` | 28080 | 主后端端口 |
| `UAS_PORT` | 8081 | UAS 后端端口 |
| `UAS_FE_PORT` | 8082 | UAS 前端端口 |

## 常用操作

```bash
# 构建并启动所有服务
docker-compose up -d

# 构建并启动特定服务
docker-compose up -d uas-app

# 重新构建并启动
docker-compose up -d --build

# 停止所有服务
docker-compose down

# 查看日志
docker-compose logs -f
docker-compose logs -f uas-app
docker logs uas-app

# 进入容器
docker exec -it uas-app sh

# 查看运行中的容器
docker ps

# 查看所有容器状态
docker-compose ps
```

## 常见问题

### Q: 端口被占用怎么办？

使用环境变量覆盖默认端口：

```bash
PORT=28081 UAS_PORT=8083 UAS_FE_PORT=8084 docker-compose up -d
```

### Q: 如何修改数据库密码？

```bash
DB_PASSWORD=your_password docker-compose up -d
```

### Q: 容器起不来怎么办？

```bash
# 查看详细日志
docker-compose logs

# 查看特定服务日志
docker-compose logs uas-app

# 重新构建
docker-compose up -d --build
```

### Q: UAS 前端页面空白？

确认 `uas/frontend/dist/` 目录下存在构建产物。如果没有，需要先构建前端：

```bash
cd uas/frontend
npm install
npm run build
```

## 公网访问

部署完成后，服务通过以下地址对外提供：

| 服务 | 地址 |
|------|------|
| 主后端 API | `http://<服务器IP>:28080` |
| UAS 后端 API | `http://<服务器IP>:8081/api/health` |
| UAS 管理界面 | `http://<服务器IP>:8082` |

> 注意：确保服务器防火墙已开放对应端口。
