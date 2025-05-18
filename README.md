# Git Repository Watcher

一个用Go语言实现的服务，用于定时检查Git仓库更新并提供Webhook功能。

## 功能特点

- 定时检查Git仓库更新
- 检查子仓库的新提交并自动更新
- 接收Webhook调用触发检查
- 在更新完成后发送Webhook通知
- 提供HTTP API查询服务状态
- 接收Webhook调用提供制品库更新功能


## 项目结构

```
├── cmd/
│   └── server/           # 服务入口点
├── configs/              # 配置文件
├── internal/             # 内部包
│   ├── git/              # Git操作相关功能
│   ├── scheduler/        # 定时调度功能
│   └── webhook/          # Webhook处理功能
├── go.mod                # Go模块文件
├── go.sum                # 依赖校验文件
└── README.md             # 项目说明文档
```

## 安装

```bash
# 克隆仓库
git clone https://github.com/Jieay/git-watcher.git
cd git-watcher

# 获取依赖
go mod tidy

# 构建
go build -o git-watcher ./cmd/server
```

## 配置

配置支持两种方式：配置文件和环境变量。**环境变量的优先级高于配置文件**。

### 配置文件

配置文件位于`configs/config.json`，包含以下主要配置项：

```json
{
  "server": {
    "port": 8080
  },
  "git": {
    "mainRepo": {
      "url": "https://github.com/example/main-repo.git",
      "branch": "main",
      "directory": "main-repo",
      "auth": {
        "type": "basic",
        "username": "your-username",
        "password": "your-password"
      }
    },
    "workingDir": "./repos",
    "useSubmodules": true,
    "branches": ["main", "develop", "release"]
  },
  "webhook": {
    "callbackUrl": "https://example.com/webhook",
    "secret": "your-webhook-secret"
  },
  "schedule": {
    "checkInterval": 600000000000
  },
  "artifactsRepo": {
    "url": "制品仓库地址",
    "branch": "默认分支名称",
    "directory": "本地目录",
    "autoBranchName": "自动合并的目标分支名称",
    "useMainAuth": true,
    "useMainCommit": true,
    "commitConfig": {
      "userName": "提交用户名",
      "userEmail": "提交邮箱",
      "message": "提交信息前缀"
    },
    "auth": {
      "type": "认证类型",
      "username": "用户名",
      "password": "密码",
      "sshPrivateKey": "SSH私钥",
      "sshKeyPath": "SSH密钥路径"
    }
  }
}
```

### 使用 SSH 密钥认证

还可以使用 SSH 密钥进行认证（configs/config.ssh.json）:

```json
{
  "git": {
    "mainRepo": {
      "url": "git@github.com:example/main-repo.git",
      "branch": "main",
      "directory": "main-repo",
      "auth": {
        "type": "ssh",
        "sshKeyPath": "/path/to/your/private_key"
      }
    },
    "useSubmodules": true
  }
}
```

### 环境变量配置

所有配置都可以通过环境变量进行设置，环境变量拥有更高的优先级：

| 配置项 | 环境变量 | 类型 | 说明 |
|--------|----------|------|------|
| 服务器端口 | `GIT_WATCHER_SERVER_PORT` | 整数 | HTTP服务器端口 |
| 主仓库URL | `GIT_WATCHER_MAIN_REPO_URL` | 字符串 | Git仓库URL |
| 主仓库分支 | `GIT_WATCHER_MAIN_REPO_BRANCH` | 字符串 | Git仓库默认分支 |
| 主仓库目录 | `GIT_WATCHER_MAIN_REPO_DIRECTORY` | 字符串 | 本地保存目录名 |
| 工作目录 | `GIT_WATCHER_WORKING_DIR` | 字符串 | 仓库工作目录 |
| 使用子模块 | `GIT_WATCHER_USE_SUBMODULES` | 布尔值 | 是否使用子模块 |
| 分支列表 | `GIT_WATCHER_BRANCHES` | 字符串 | 需检查的分支，逗号分隔 |
| 认证类型 | `GIT_WATCHER_AUTH_TYPE` | 字符串 | "none", "basic", "ssh" |
| 用户名 | `GIT_WATCHER_AUTH_USERNAME` | 字符串 | Git认证用户名 |
| 密码 | `GIT_WATCHER_AUTH_PASSWORD` | 字符串 | Git认证密码 |
| SSH密钥路径 | `GIT_WATCHER_AUTH_SSH_KEY_PATH` | 字符串 | SSH私钥文件路径 |
| SSH私钥 | `GIT_WATCHER_AUTH_SSH_PRIVATE_KEY` | 字符串 | SSH私钥内容 |
| 自动提交 | `GIT_WATCHER_AUTO_COMMIT` | 布尔值 | 是否自动提交 |
| 仓库提交用户名 | `GIT_WATCHER_COMMIT_USER_NAME` | 字符串 | 仓库Git提交用户名 |
| 仓库提交邮箱 | `GIT_WATCHER_COMMIT_USER_EMAIL` | 字符串 | 仓库Git提交邮箱 |
| 仓库提交信息 | `GIT_WATCHER_COMMIT_MESSAGE` | 字符串 | 仓库Git提交信息前缀 |
| Webhook回调URL | `GIT_WATCHER_WEBHOOK_CALLBACK_URL` | 字符串 | 更新后回调的URL |
| Webhook密钥 | `GIT_WATCHER_WEBHOOK_SECRET` | 字符串 | Webhook安全密钥 |
| Webhook请求方法 | `GIT_WATCHER_WEBHOOK_METHOD` | 字符串 | HTTP请求方法(GET/POST) |
| 检查间隔 | `GIT_WATCHER_CHECK_INTERVAL` | 整数/时间 | 定时检查间隔，可以是纳秒数或时间格式(例如：10m) |
| 制品仓库URL | `GIT_WATCHER_ARTIFACTS_REPO_URL` | 字符串 | 制品仓库地址 |
| 制品仓库分支 | `GIT_WATCHER_ARTIFACTS_REPO_BRANCH` | 字符串 | 制品仓库默认分支 |
| 制品仓库目录 | `GIT_WATCHER_ARTIFACTS_REPO_DIRECTORY` | 字符串 | 制品仓库本地目录 |
| 制品仓库自动合并分支 | `GIT_WATCHER_ARTIFACTS_REPO_AUTO_BRANCH` | 字符串 | 自动合并的目标分支名称 |
| 制品仓库使用主仓库认证 | `GIT_WATCHER_ARTIFACTS_USE_MAIN_AUTH` | 布尔值 | 是否使用主仓库的认证信息 |
| 制品仓库使用主仓库提交配置 | `GIT_WATCHER_ARTIFACTS_USE_MAIN_COMMIT` | 布尔值 | 是否使用主仓库的提交信息配置 |
| 制品仓库认证类型 | `GIT_WATCHER_ARTIFACTS_AUTH_TYPE` | 字符串 | 认证类型（"none", "basic", "ssh"） |
| 制品仓库用户名 | `GIT_WATCHER_ARTIFACTS_AUTH_USERNAME` | 字符串 | 制品仓库认证用户名 |
| 制品仓库密码 | `GIT_WATCHER_ARTIFACTS_AUTH_PASSWORD` | 字符串 | 制品仓库认证密码 |
| 制品仓库SSH密钥路径 | `GIT_WATCHER_ARTIFACTS_AUTH_SSH_KEY_PATH` | 字符串 | 制品仓库SSH私钥文件路径 |
| 制品仓库SSH私钥 | `GIT_WATCHER_ARTIFACTS_AUTH_SSH_PRIVATE_KEY` | 字符串 | 制品仓库SSH私钥内容 |
| 制品仓库提交用户名 | `GIT_WATCHER_ARTIFACTS_COMMIT_USERNAME` | 字符串 | 制品仓库Git提交用户名 |
| 制品仓库提交邮箱 | `GIT_WATCHER_ARTIFACTS_COMMIT_EMAIL` | 字符串 | 制品仓库Git提交邮箱 |
| 制品仓库提交信息 | `GIT_WATCHER_ARTIFACTS_COMMIT_MESSAGE` | 字符串 | 制品仓库Git提交信息前缀 |

### 配置项说明

- `git.mainRepo.auth`: 认证配置（basic 或 ssh）
- `git.useSubmodules`: 是否使用子模块（为 true 时自动处理 .gitmodules）
- `git.branches`: 定时任务需要检查的分支列表
- `git.workingDir`: 仓库工作目录
- `webhook.callbackUrl`: 更新完成后通知的Webhook URL
- `webhook.secret`: Webhook安全密钥
- `schedule.checkInterval`: 检查间隔时间（可以是纳秒整数值或时间字符串如"10m"）
- `artifactsRepo`: 制品仓库配置
  - `url`: 制品仓库地址
  - `branch`: 默认分支名称
  - `directory`: 本地工作目录
  - `autoBranchName`: 自动合并的目标分支名称，如果不设置则使用 `branch` 字段的值
  - `useMainAuth`: 是否使用主仓库的认证信息
  - `useMainCommit`: 是否使用主仓库的提交信息配置
  - `commitConfig`: 提交信息配置
    - `userName`: Git 提交用户名
    - `userEmail`: Git 提交邮箱
    - `message`: 提交信息前缀，会与时间、仓库名、包名和版本信息组合
  - `auth`: 认证配置（当 `useMainAuth` 为 false 时使用）
    - `type`: 认证类型（"none", "basic", "ssh"）
    - `username`: 用户名（basic 认证）
    - `password`: 密码（basic 认证）
    - `sshPrivateKey`: SSH 私钥内容
    - `sshKeyPath`: SSH 密钥文件路径

#### 提交信息格式

制品仓库的提交信息会按照以下格式生成：

```
[commitConfig.message]

Time: [时间戳]
ArtifactRepoName: [仓库名]
Package: [包名]
Version: [版本号]
```

## 使用方法

### 直接运行

```bash
# 使用默认配置文件启动服务
./git-watcher

# 指定配置文件
./git-watcher -config=/path/to/your/config.json
```

### Docker 方式运行

项目提供 Dockerfile 和 docker-compose.yml 文件，方便使用 Docker 部署。

```bash
# 构建 Docker 镜像
docker build -t git-watcher .

# 运行容器
docker run -d -p 8080:8080 -v $(pwd)/configs/config.json:/app/configs/config.json -v git-repos:/app/repos --name git-watcher git-watcher

# 或者使用 Docker Compose
docker-compose up -d

# 使用环境变量配置（不需要配置文件）
docker run -d -p 8080:8080 \
  -e GIT_WATCHER_MAIN_REPO_URL="https://github.com/example/main-repo.git" \
  -e GIT_WATCHER_MAIN_REPO_BRANCH="main" \
  -e GIT_WATCHER_AUTH_TYPE="basic" \
  -e GIT_WATCHER_AUTH_USERNAME="your-username" \
  -e GIT_WATCHER_AUTH_PASSWORD="your-password" \
  -e GIT_WATCHER_USE_SUBMODULES="true" \
  -e GIT_WATCHER_BRANCHES="main,develop" \
  -e GIT_WATCHER_CHECK_INTERVAL="10m" \
  -v git-repos:/app/repos --name git-watcher git-watcher
```

## API接口

### 健康检查

```
GET /health
```

返回服务健康状态。

### 触发手动检查

```
POST /webhook/trigger
```

触发仓库检查。可以通过Webhook调用此接口。

#### 请求头

```
Content-Type: application/json
X-Webhook-Signature: 2a55955309a33633b5d67ae35f8669d77cddf078e9b28bb632357468be073824  // 可选，如配置了密钥则必须
```

#### 请求体格式

```json
{
  "event": "push",
  "branch": "main",       // 可选，指定要检查的分支
  "reference": "refs/heads/develop"  // 可选，Git引用，会自动提取分支名
}
```

#### 请求参数说明

- `event`: 事件类型，任意字符串，用于日志记录
- `branch`: 要检查的分支名称，如果提供此参数，将只检查该分支
- `reference`: Git引用格式，如 "refs/heads/develop"，系统会自动提取分支名

#### 行为说明

- 如果请求中包含 `branch` 参数，则只检查指定的分支
- 如果请求中包含 `reference` 参数（如 GitHub webhook 的格式），会自动提取分支名
- 如果未指定分支，则检查所有配置的分支

#### 签名验证

如果在配置中设置了 `webhook.secret`，请求必须包含签名。签名生成方法：

```
HMAC-SHA256(请求体, webhook.secret)
```

将生成的十六进制字符串放在 `X-Webhook-Signature` 请求头中。

#### 请求示例

1. **检查特定分支**

```bash
curl -X POST http://localhost:8080/webhook/trigger \
  -H "Content-Type: application/json" \
  -d '{"event":"manual","branch":"main"}'
```

2. **使用 Git 引用格式（适用于 GitHub Webhook）**

```bash
curl -X POST http://localhost:8080/webhook/trigger \
  -H "Content-Type: application/json" \
  -d '{"event":"push","reference":"refs/heads/develop"}'
```

3. **带签名验证的请求**

```bash
curl -X POST http://localhost:8080/webhook/trigger \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: 2a55955309a33633b5d67ae35f8669d77cddf078e9b28bb632357468be073824" \
  -d '{"event":"push","branch":"main"}'
```

4. **检查所有分支**

```bash
curl -X POST http://localhost:8080/webhook/trigger \
  -H "Content-Type: application/json" \
  -d '{"event":"manual"}'
```

#### 响应示例

成功响应:
```
Manual check for branch main completed
```
或
```
Manual check for all branches triggered
```

错误响应:
```json
{
  "error": "Invalid webhook signature"
}
```

### 服务状态

```
GET /status
```

返回调度器当前状态。

## 安全性

- Webhook通信使用HMAC-SHA256签名验证
- 配置文件中的敏感信息应妥善保管

## 许可证

MIT 

