# NOFX 离线认证与敏感数据加密说明

## 文档信息
- 更新时间：2026-04-25
- 状态：已按当前代码实现更新
- 适用范围：本地部署、Docker 部署、后端认证链路、敏感配置存储链路

## 改造目标
- 将原有账号密码和云端依赖认证收敛为单实例管理员密钥登录。
- 使用 RS256 JWT 管理访问令牌与刷新令牌，支持会话吊销和一次性刷新令牌轮换。
- 使用 RSA-OAEP 包装 AES-256-GCM 数据密钥，保证所有敏感配置密文落盘。
- 保留旧认证、旧 onboarding、旧浏览器加密兼容文件，默认停用主路径，确保可回滚。

## 当前实现架构

### 1. 本地管理员密钥登录
- 首次启动时自动生成管理员登录密钥，只在控制台或容器日志中输出一次。
- 数据库仅保存管理员密钥的 bcrypt 哈希，不保存明文。
- 登录接口固定为：
  - `POST /api/auth/login`
  - `POST /api/auth/refresh`
  - `POST /api/auth/logout`
  - `GET /api/auth/status`

### 2. RS256 会话体系
- 启动时自动生成或加载本地 RSA 4096 根密钥对。
- `access_token` 默认有效期 2 小时。
- `refresh_token` 默认有效期 7 天，并且每次刷新后立即轮换。
- 会话信息持久化到 `auth_sessions` 表，支持登出、密钥重置、版本变更后立刻失效。

### 3. 根密钥与数据密钥
- 根密钥目录默认位于 `config/keys/`。
- Unix 平台会收紧到目录 `0700`、私钥文件 `0600`。
- Windows 平台会尝试收紧为当前用户可访问的 ACL。
- AES 数据密钥仅在内存中保持明文，数据库中只保存 RSA-OAEP 加密后的包装值。

### 4. 敏感数据落盘加密
- 以下数据改为 AES-256-GCM 密文落盘：
  - 交易所 API Key、Secret、Passphrase、钱包私钥等
  - AI 模型 API Key
  - Telegram Bot Token
  - 策略配置中的嵌套敏感字段
- 刷新令牌不以可逆密文存储，仅保存哈希。
- 前端不再负责主路径加密，敏感值提交后立即由后端加密存储。

### 5. 兼容层处理
- 旧账号密码认证处理函数仍然保留，但默认不注册到路由。
- 旧 onboarding / wallet 自动钱包处理函数仍然保留，但只返回停用提示。
- `web/src/lib/crypto.ts` 继续保留为兼容文件，但已经退出主路径。

## 首次启动与日常使用

### 手动部署
```bash
go run main.go
```

首次启动时需要关注：
- 控制台输出的管理员登录密钥
- `config/keys/` 根密钥目录路径
- 根公钥指纹
- 备份提醒

建议在首次启动后立即完成：
1. 备份 `config/keys/`
2. 备份 `backup/`
3. 确认旧 `.env` 中的明文敏感项已经迁移，再手动清理旧文件

### Docker 部署
```bash
docker compose up -d
docker compose logs -f nofx
```

默认需要持久化以下目录：
- `./config/keys:/app/config/keys`
- `./backup:/app/backup`
- `./data:/app/data`

如果删除容器前未保留 `config/keys/`，已经加密的敏感数据将无法解密恢复。

## 运维命令

### 重置管理员密钥
```bash
./nofx reset-admin-key
```

Docker：
```bash
docker compose exec nofx ./nofx reset-admin-key
```

作用：
- 生成新的管理员登录密钥
- 更新数据库中的 bcrypt 哈希
- 提升认证版本并使旧会话失效

### 重置根密钥
```bash
./nofx reset-root-key
```

Docker：
```bash
docker compose exec nofx ./nofx reset-root-key
```

作用：
- 备份数据库和旧根密钥
- 生成新的 RSA 根密钥对
- 重新包装 AES 数据密钥
- 重新加密现有敏感数据
- 使旧会话失效

### 恢复备份
```bash
./nofx restore-backup <timestamp>
```

Docker：
```bash
docker compose exec nofx ./nofx restore-backup <timestamp>
```

作用：
- 按备份清单恢复数据库
- 恢复根密钥目录
- 回滚迁移前配置快照

## 数据迁移策略
- 启动时会先检查旧数据库、旧 `.env`、旧 `config.json` 中是否存在明文敏感项。
- 迁移前自动写入 `backup/<timestamp>/`。
- 迁移完成后，接口仅返回 `configured`、掩码摘要、更新时间等状态字段，不再回显明文或密文。

## 回滚说明
- 快速回滚优先使用 `restore-backup` 恢复到迁移前状态。
- 旧认证和旧 onboarding 代码文件仍然保留，便于在明确评估后进行定点回滚。
- 不建议手工回滚数据库与根密钥目录的单个文件，应以同一批次备份为单位恢复。

## 变更影响范围
- 后端：`main.go`、`bootstrap`、`auth`、`crypto`、`store`、`api`
- 前端：`AuthContext`、`httpClient`、登录页、设置页、兼容 onboarding 页面
- 部署：`docker-compose.yml`、`start.sh`、`.gitignore`

## 已废弃但保留的旧路径
- 旧账号密码认证接口处理函数
- 旧 onboarding 自动钱包流程
- 浏览器端 `/api/crypto/public-key` 与 `/api/crypto/decrypt` 主路径依赖

保留原因：
- 便于兼容旧版本代码结构
- 便于必要时快速回滚
- 避免一次性删除大量旧代码造成不可控影响
