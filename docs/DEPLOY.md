# 部署文档（BYOK 单门面）

> 仓库：https://github.com/mimaweimahan/tongyiapi  
> 演示：https://www.magicplus.cc  
> 更新：2026-11-04  
> 许可：AGPL-3.0（基于 QuantumNous/new-api）

本文说明如何把**本仓库源码**（含 BYOK 改造）部署成可对外的单门面站点。  
**不要**直接拉 `calciumion/new-api:latest` 官方镜像当生产——官方镜像**没有**本仓的 BYOK 能力，必须用本仓库 `Dockerfile` 构建。

---

## 0. 一分钟架构

```
用户 / 客户端
  → https://你的域名
      → Nginx / Caddy / Cloudflare
          → 本服务 :3000（容器或二进制）
              ├─ Web 控制台（注册、/my-keys、令牌、价表、指南…）
              └─ OpenAI 兼容 /v1（含 model=auto）
```

数据默认可落 SQLite（单机最简单）；生产建议 PostgreSQL + Redis（见 Compose）。

---

## 1. 环境要求

| 项 | 最低建议 |
|---|---|
| OS | Linux x86_64 / arm64 |
| Docker | 24+，含 `docker compose` 插件 |
| 或源码构建 | Go **1.26+**、Bun **1.4.4**、git |
| 内存 | 构建期 ≥ 4GB；运行 SQLite 单机 ≥ 1GB |
| 磁盘 | 构建镜像约需数 GB 临时空间 |
| 域名 | 生产务必 HTTPS；Cloudflare 橙云时 SSL 建议 **Full** |

---

## 2. 一键部署（推荐 · Docker 构建本仓）

在目标机器执行：

```bash
curl -fsSL https://raw.githubusercontent.com/mimaweimahan/tongyiapi/main/docs/one-click-deploy.sh | bash
```

或手动：

```bash
git clone https://github.com/mimaweimahan/tongyiapi.git
cd tongyiapi
chmod +x docs/one-click-deploy.sh
./docs/one-click-deploy.sh
```

脚本会：

1. 生成 `SESSION_SECRET`  
2. 用本仓 `Dockerfile` 构建镜像（**含 BYOK**）  
3. 启动 `docs/docker-compose.byok.yml`（SQLite + 本机 `:3000`）  
4. 打印访问地址与首次超管向导提示  

首次打开浏览器完成**安装向导**（创建 Root 超管）。  
然后到系统设置里把 **Server Address** 设成你的公网 HTTPS 地址（影响 sitemap / 部分外链）。

### 常用命令

```bash
cd tongyiapi   # 或脚本打印的目录
docker compose -f docs/docker-compose.byok.yml ps
docker compose -f docs/docker-compose.byok.yml logs -f new-api
docker compose -f docs/docker-compose.byok.yml restart new-api
docker compose -f docs/docker-compose.byok.yml down
```

数据目录默认：`./data`（SQLite 等），请纳入备份。

---

## 3. 生产 Compose（PostgreSQL + Redis）

适合多用户 / 要持久化与缓存的场景。

1. 复制并改密码：

```bash
cp docs/docker-compose.prod.example.yml docker-compose.prod.yml
# 编辑：POSTGRES 密码、REDIS 密码、SESSION_SECRET、端口
```

2. 构建并启动：

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml logs -f new-api
```

3. 反代到 `127.0.0.1:3000`，开启 HTTPS，并配置：

| 变量 / 配置 | 说明 |
|---|---|
| `SESSION_SECRET` | 随机长串；多节点必须一致 |
| `SESSION_COOKIE_SECURE=true` | 仅 HTTPS 控制台 |
| `SESSION_COOKIE_TRUSTED_URL` | 精确 Origin，如 `https://www.example.com` |
| `TRUSTED_PROXIES` | 反代网段或 Cloudflare 回源 IP |
| 反向代理 | 关闭或拉长流式超时；支持 WebSocket Upgrade |

### Cloudflare 注意

- 访客 HTTPS + 源站若只开 HTTP：可用 Flexible，但**源站不要再 301 强制跳 HTTPS**（会环）。  
- 源站已挂证书时：CF SSL 选 **Full**（或 Full strict）。  
- Let’s Encrypt 可用 webroot：`/.well-known/acme-challenge/`。

参考现网 nginx 要点（宝塔同理）：

```nginx
location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "";
    proxy_read_timeout 300s;
    proxy_buffering off;
}
```

---

## 4. 源码直接编译（无 Docker）

```bash
export PATH="$HOME/.bun/bin:/usr/local/go/bin:$PATH"
git clone https://github.com/mimaweimahan/tongyiapi.git
cd tongyiapi

cd web && bun install --frozen-lockfile && bun run build && cd ..
go build -o new-api .

mkdir -p data logs
PORT=3000 TZ=Asia/Shanghai ./new-api
# 或: PORT=3000 ./new-api --log-dir ./logs
```

systemd 示例见 `docs/new-api.service.example`。

---

## 5. 上线检查清单

- [ ] 打开站点能进安装向导 / 登录，不是空白或 502  
- [ ] Root 可进「BYOK 厂商」目录（侧栏）  
- [ ] 普通用户注册后 `group` 为 `u{id}`（库内或后续绑钥验证）  
- [ ] `/my-keys` 能保存至少一家官方钥；`/test` 探活  
- [ ] 创建本站令牌，客户端 Base URL = `https://域名/v1`，Key = `sk-...`  
- [ ] `model=auto` 能出字；响应头可见 `X-Byok-Model` / `X-Byok-Tried`（按实现）  
- [ ] 个人 BYOK 渠道调用**不扣**本站 `quota`（真钱走厂商账单）  
- [ ] `Server Address` = 公网 HTTPS  
- [ ] 备份：`data/` 或数据库卷；回滚预案写明  

功能说明见 [`FEATURES.md`](./FEATURES.md)。

---

## 6. 升级

```bash
cd tongyiapi
git pull
docker compose -f docs/docker-compose.byok.yml up -d --build
# 或 prod compose 同理
```

升级前**备份数据库**。大版本先看上游 New API Release notes。

---

## 7. 回滚

- Docker：保留上一镜像 tag，`docker compose ... up -d` 切回；数据卷勿删错。  
- 二进制：保留旧 `new-api` 文件与对应 `data` 备份。  
- 现网若仍保留对照端口（如旧门户），仅摘域名反代即可切回。

---

## 8. 合规与售卖边界（必读）

- 本项目基于 New API，**AGPL-3.0**：分发修改版须履行 AGPL 义务。  
- 商业上建议：卖**部署 / 代装 / 换皮 / 运维**，勿把「闭源魔改货盘」当卖点。  
- 用户须自行合法取得各厂商 API Key，并遵守上游与当地法规。

---

## 9. 故障速查

| 现象 | 排查 |
|---|---|
| 构建失败 bun/go | 内存不足；核对 Dockerfile 内 Go/Bun 版本 |
| 502 | 容器未起；`docker compose logs`；端口占用 |
| CF 重定向环 | Flexible + 源站强制 HTTPS → 去掉源站 80→443 强制跳 |
| 登录后踢下线 | 勿在登录后改用户 group；本仓注册应已写入 `u{id}` |
| auto 打到别的厂 | 令牌是否「厂商+auto」并锁了该厂模型列表 |
| 官方镜像无 BYOK | 必须用本仓 Dockerfile 构建，勿用 `calciumion/new-api:latest` 当本产品 |
