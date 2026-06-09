# 校园二手交易平台

Go + MySQL 构建的校园二手交易平台，支持用户注册登录、商品发布浏览、订单管理等完整交易流程，管理员可管理数据库。

## 技术栈

| 层级   | 技术                 |
| ------ | -------------------- |
| 前端   | 原生 HTML + CSS + JS |
| 后端   | Go + Gin 框架        |
| 数据库 | MySQL 8.0            |
| 认证   | SSO + JWT            |
| 部署   | Docker / Nginx       |

---

## 完整部署指南（Ubuntu / CentOS）

### 1. 安装依赖

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y        # Ubuntu
# sudo yum update -y                           # CentOS

# 安装 Docker（推荐）
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# 退出重新登录，或执行: newgrp docker

# 安装 Git
sudo apt install -y git                         # Ubuntu
# sudo yum install -y git                       # CentOS

# 安装 Nginx
sudo apt install -y nginx                       # Ubuntu
# sudo yum install -y nginx                     # CentOS
sudo systemctl enable nginx
sudo systemctl start nginx
```

### 2. 克隆项目

```bash
git clone https://github.com/FuwaMintNEKO/Gongchengshijian.git school-trade
cd school-trade
```

### 3. 使用 Docker Compose 启动（自动创建数据库）

```bash
# 启动（MySQL + 应用）
docker compose up -d

# 查看日志
docker compose logs -f app

# 验证健康检查
curl http://localhost:28080/health
# 预期: {"database":true,"status":"ok","time":"..."}
```

> 无需手动安装 MySQL！`docker-compose.yml` 会自动创建 MySQL 8.0 容器和服务数据库 `school_trade`，应用启动时会自动建表并插入示例数据。

**环境变量说明（可选）：**

| 变量         | 默认值     | 说明           |
| ------------ | ---------- | -------------- |
| `PORT`       | `28080`    | 应用端口       |
| `DB_PASSWORD`| `114514`   | MySQL 密码     |
| `DB_NAME`    | `school_trade` | 数据库名   |

```bash
# 自定义启动示例
DB_PASSWORD=MySecurePwd DB_NAME=myschool docker compose up -d
```

### 4. 配置 Nginx 反向代理

创建 Nginx 配置文件：

```bash
sudo nano /etc/nginx/sites-available/school-trade
# 或 Ubuntu:  /etc/nginx/conf.d/school-trade.conf
```

写入以下内容：

```nginx
server {
    listen 80;
    server_name your-domain.com;  # 改成你的域名或服务器 IP

    # 后端 API + 静态文件（由 Go 服务统一处理）
    location / {
        proxy_pass http://127.0.0.1:28080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

启用并重载 Nginx：

```bash
# Ubuntu
sudo ln -s /etc/nginx/sites-available/school-trade /etc/nginx/sites-enabled/
sudo nginx -t          # 测试配置是否正确
sudo systemctl reload nginx   # 重载 Nginx

# CentOS
sudo nginx -t
sudo systemctl reload nginx
```

### 5. 开放防火墙端口

```bash
# Ubuntu (ufw)
sudo ufw allow 80/tcp
sudo ufw allow 22/tcp
sudo ufw --force enable

# CentOS (firewalld)
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --reload
```

### 6. 访问验证

- 浏览器访问 `http://your-domain.com` 或 `http://服务器IP`
- 默认管理员账号：`admin` / `admin123`
- 测试用户：`alice` / `alice123`、`bob` / `bob123`

### 7. 配置 HTTPS（推荐）

使用 Let's Encrypt 免费证书：

```bash
sudo apt install -y certbot python3-certbot-nginx   # Ubuntu
# sudo yum install -y certbot python3-certbot-nginx # CentOS

sudo certbot --nginx -d your-domain.com
```

Certbot 会自动修改 Nginx 配置并开启 HTTPS。

### 8. 日常维护

```bash
# 查看服务状态
docker compose ps

# 查看应用日志
docker compose logs -f app

# 重启应用
docker compose restart app

# 更新代码后重新部署
git pull
docker compose build app
docker compose up -d

# 停止所有服务
docker compose down

# 备份数据库
docker compose exec -T mysql mysqldump -uroot -p114514 school_trade > backup.sql
```

---

## 非 Docker 手动部署

如果不使用 Docker，需自行安装 MySQL 8.0+ 和 Go 1.24+，并配置环境变量：

```bash
# 设置数据库环境变量
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=你的密码
export DB_NAME=school_trade
export PORT=28080

# 手动创建数据库
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS school_trade DEFAULT CHARSET utf8mb4;"

# 编译并运行
cd backend
go build -o school-trade .
./school-trade
```

## 项目结构

```
school-trade/
├── backend/              # Go 后端
│   ├── handlers/         # 接口处理
│   ├── middleware/       # JWT 认证中间件
│   ├── models/           # 数据模型
│   └── store/            # 数据库层
├── frontend/             # 前端页面
│   ├── css/
│   ├── js/
│   └── pages/
├── docker-compose.yml    # Docker 编排
├── Dockerfile            # 应用 Dockerfile
└── README.md
```
