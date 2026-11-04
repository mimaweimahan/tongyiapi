# 功能文档（BYOK 单门面）

> 仓库：https://github.com/mimaweimahan/tongyiapi  
> 演示：https://www.magicplus.cc  
> 更新：2026-11-04  
> 对齐实现：VPS `byok-single-facade` @ `7f5fbc1`（P1–P15）

本文面向**站长与终端用户**：说明本站「自带官方钥 → 一把本站令牌」能做什么、不能做什么。  
部署步骤见 [`DEPLOY.md`](./DEPLOY.md)。

---

## 1. 产品一句话

用户把自己的**官方厂商 API Key**交给本站 → 得到：

- 一个控制台（注册、密钥、令牌、日志、价表、指南）  
- 一个 OpenAI 兼容入口：`Base URL = https://你的域名/v1`  
- 一把**本站令牌**（`sk-...`）

**钱走各厂商官方账单**；本站不是上游货盘。个人 BYOK 渠道调用**不扣本站额度钱包**（仍可记用量日志）。

---

## 2. 角色与入口

| 角色 | 主要入口 | 能力 |
|---|---|---|
| 普通用户 | `/my-keys`、令牌创建、控制台日志 | 绑自己的官方钥、建令牌、调 `/v1` |
| 超管 Root | 侧栏「BYOK 厂商」、权益、系统设置 | 配置厂商目录、免费/收费、开通权益、Server Address |
| 客户端 | Cursor / 鑫源 / OpenClaw / 任意 OpenAI SDK | 只配本站 Base URL + 本站令牌 |

---

## 3. 用户功能

### 3.1 账号与隔离

- 注册 / 登录使用本站用户体系。  
- 新用户自动分到个人组：`group = u{用户ID}`。  
- 个人渠道命名：`u{id}-{provider}`（如 `u15-silicon`）。  
- 本站令牌绑定同一 `u{id}`，与他人渠道隔离。

### 3.2 我的密钥 `/my-keys`

- 按**超管启用的厂商目录**展示卡片（默认种子含常见官方厂：OpenAI、Azure、Anthropic、Gemini、DeepSeek、阿里、月之暗面、智谱、百度、腾讯、火山、硅基、MiniMax 等）。  
- 保存 / 更新 / 删除自己的官方 Key。  
- 「从厂商拉取模型」、填写前/保存前**探活**（`/api/my-keys/test`）。  
- 厂商官方取钥链接（若目录配置了 `key_url`）。  
- **免费**厂商：可直接填钥。  
- **收费**厂商：须超管开通权益后才能填钥（开通后仍是用户自己的钥，**不是**站长共享钥）。

### 3.3 本站令牌

在 `/my-keys`（或令牌相关入口）创建：

| 选项 | 行为 |
|---|---|
| 指定厂商 + 具体模型 | 令牌只能打该模型 |
| 指定厂商 + `auto` | 锁定该厂对话模型列表（避免串到别的厂） |
| 全部已接入 + `auto` | 不锁模型列表，可在已接入厂间按路由策略尝试 |
| 返回 | `base_url` + 明文 `sk-...`（只展示一次） |

令牌一般为无限本站额度标志；**真消费在官方**。个人 BYOK 路径跳过本站钱包预扣/结算。

### 3.4 统一 API 与 `auto`

| 项 | 说明 |
|---|---|
| 协议 | OpenAI 兼容：`POST /v1/chat/completions` 等 |
| 模型列表 | `GET /v1/models` 注入 `auto` / `auto-chat` 置顶 |
| `auto` 路由 | 券池优先 → 余额池；按对话意图重排（视觉/推理/代码/工具/长文/闲聊） |
| Failover | 对 `429/503/404` 及部分业务错误换模型重试 |
| 观测 | 响应头如 `X-Byok-Model`、`X-Byok-Tried`、`X-Byok-Intent` |

注意：客户端里的 `auto` ≠ Cursor 顶栏「Auto」（那是 Cursor 云端）。

### 3.5 价表与用量估价

- 公开/站内价表页：按厂商展示参考价（如硅基、Gemini 等数据源）。  
- 用量估价：结合日志 token × 价表估算官方费用（币种分列，不硬兑）。  
- **估价≠扣费**；扣费以各厂商账单为准。

### 3.6 指南与 SEO

- `/guides` 接入指南（Cursor、OpenClaw、硅基、BYOK、auto 等种子内容）。  
- `/robots.txt`、`/sitemap.xml`（密钥类路径应 Disallow）。  
- 系统 **Server Address** 影响站点基址。

---

## 4. 超管功能

### 4.1 BYOK 厂商目录

- CRUD：名称、渠道类型、排序、启用/停用、取钥链接、`pricing_tier`（`free` / `paid`）。  
- 用户 `/my-keys` **只读启用中**的目录项。  
- **不是**往渠道里塞共享官方钥；只是「允许用户绑定哪些厂」。

### 4.2 权益（收费厂商）

- 按用户开通 / 撤销某 `provider` 的填钥资格。  
- 未开通：禁止保存/探活/拉模。  
- 撤销后再次禁止；已存在渠道是否清理由运维策略决定（默认以门禁为准）。

### 4.3 系统与运维（继承 New API）

- 渠道 / 用户 / 日志 / 主题 / OAuth 等原版能力仍在。  
- 站长**共享渠道**（非 `u{id}-*` 个人渠）仍走原额度逻辑。  
- 建议生产配置 HTTPS、备份库、限制注册或配合审核。

---

## 5. 客户端怎么接

| 项 | 值 |
|---|---|
| Base URL | `https://你的域名/v1` |
| API Key | 本站令牌 `sk-...` |
| Model | `auto` 或官方具体模型名 |
| 示例 | Cursor / 鑫源自定义 Provider；OpenAI SDK `base_url` 指向本站 |

```bash
curl -sS https://你的域名/v1/models \
  -H "Authorization: Bearer sk-你的本站令牌"

curl -sS https://你的域名/v1/chat/completions \
  -H "Authorization: Bearer sk-你的本站令牌" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"你好"}]}'
```

---

## 6. 明确不做 / 边界

| 不做 | 说明 |
|---|---|
| 伪装 OpenLux 等中转当「官方厂」 | 目录名与文案须诚实 |
| 默认卖共享官方钥货盘 | 收费=开通后用户自填钥 |
| 订阅拆官方钥 / 指纹隐藏 | 产品范围外 |
| 扣用户本站余额当「官方费用」 | 个人 BYOK 跳过站点钱包 |
| 两套站（门户+第二控制台） | 目标形态为**单门面** |

可增强（未默认承诺）：自定义任意 OpenAI 兼容 Base URL+Key（须标明钱走谁）。

---

## 7. 关键 API（开发对照）

| Method | Path | 说明 |
|---|---|---|
| GET/POST | `/api/my-keys/` | 列表 / 保存厂商钥 |
| DELETE | `/api/my-keys/:provider` | 删除 |
| POST | `/api/my-keys/fetch_models` | 拉模型 |
| POST | `/api/my-keys/test` | 探活（可不落库） |
| POST | `/api/my-keys/site-token` | 创建本站令牌 |
| GET/POST | `/api/byok/providers` | 超管：厂商目录 |
| GET/POST | `/api/byok/entitlements` | 超管：权益 |
| GET | `/v1/models` | 含 auto 注入 |
| POST | `/v1/chat/completions` | 含 auto 路由 |

返回协议：`{ "success": bool, "message": string, "data": ... }`。

---

## 8. 验收速查（上线后）

1. 用户注册 → 组为 `u{id}`。  
2. `/my-keys` 保存硅基（或目录内免费厂）→ 探活成功。  
3. 建令牌（厂商+auto）→ 客户端 `auto` 出字。  
4. 本站 `quota=0` 仍能走个人 BYOK（不被钱包拦住）。  
5. 超管设某厂为 paid → 未开通用户无法保存；开通后可保存。  

更细的工程对照见项目内迁移清单（若你同时维护 `byok-portal` 工作区的 `DEV_FEATURES.md`）。
