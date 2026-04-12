# Frontend API

本文档描述当前前端可直接对接的后端 HTTP 接口。

适用范围：
- 用户认证
- 注册 / 登录 / 登出
- 当前用户查询
- 多会话列表
- 会话详情
- 发送消息

当前文档只覆盖已经落地的接口，不包含未实现能力。

## Base URL

开发环境直连后端时：

```text
http://localhost:8080
```

如果前端使用 Next.js 服务端代理，则浏览器请求前端自己的 `/api/*` 即可，由前端代理转发到后端。

## 认证方式

后端使用 `HttpOnly Cookie` 维持登录态。

前端注意事项：
- 浏览器请求必须携带 cookie
- `fetch` 需要设置 `credentials: "include"`
- 前端不要自行存储认证 token

示例：

```ts
await fetch("http://localhost:8080/auth/login", {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  credentials: "include",
  body: JSON.stringify({
    email,
    password
  })
});
```

## 通用错误格式

所有错误统一返回：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "invalid request body"
  }
}
```

## 1. 认证接口

### `POST /auth/register`

注册并自动登录。

请求体：

```json
{
  "email": "user@opencumt.org",
  "password": "StrongPassword123",
  "confirm_password": "StrongPassword123",
  "display_name": "Qingke"
}
```

成功响应：
- `201 Created`

```json
{
  "success": true,
  "message": "register succeeded",
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

常见失败：
- `400 Bad Request`
  - `invalid_email_format`
  - `invalid_email_domain`
  - `password_mismatch`
  - `weak_password`
- `409 Conflict`
  - `email_already_registered`

### `POST /auth/login`

登录。

请求体：

```json
{
  "email": "user@opencumt.org",
  "password": "StrongPassword123"
}
```

成功响应：
- `200 OK`

```json
{
  "success": true,
  "message": "login succeeded",
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

失败响应：
- `401 Unauthorized`

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "invalid credentials"
  }
}
```

### `POST /auth/logout`

登出。

请求体：
- 无

成功响应：
- `200 OK`

```json
{
  "success": true,
  "message": "logout succeeded"
}
```

### `GET /auth/me`

获取当前登录用户。

请求体：
- 无

成功响应：
- `200 OK`

```json
{
  "success": true,
  "message": "current user loaded",
  "user": {
    "id": "user_xxx",
    "email": "user@opencumt.org",
    "display_name": "Qingke"
  }
}
```

未登录：
- `401 Unauthorized`

## 2. 会话接口

以下接口全部要求已登录。

### `POST /sessions`

创建新会话。

请求体：

```json
{
  "channel": "web"
}
```

当前可用 `channel`：
- `web`
- `bot`

成功响应：
- `201 Created`

```json
{
  "id": "session_xxx",
  "user_id": "user_xxx",
  "title": "新会话",
  "channel": "web",
  "history": [],
  "created_at": "2026-04-12T12:00:00Z",
  "updated_at": "2026-04-12T12:00:00Z"
}
```

### `GET /sessions`

获取当前登录用户的会话列表。

请求体：
- 无

成功响应：
- `200 OK`

```json
{
  "items": [
    {
      "id": "session_xxx",
      "title": "新会话",
      "channel": "web",
      "updated_at": "2026-04-12T12:00:00Z"
    }
  ]
}
```

### `GET /sessions/{id}`

获取当前登录用户拥有的某个会话详情。

成功响应：
- `200 OK`

```json
{
  "id": "session_xxx",
  "user_id": "user_xxx",
  "title": "新会话",
  "channel": "web",
  "history": [
    {
      "id": "user_xxx",
      "user": {
        "id": "user_xxx",
        "name": "Qingke"
      },
      "message": {
        "content": "你好"
      },
      "sequence": 1,
      "source": "user",
      "created_at": "2026-04-12T12:01:00Z"
    },
    {
      "id": "agent",
      "user": {
        "id": "agent",
        "name": "DM Agent"
      },
      "message": {
        "content": "你好，我在。"
      },
      "sequence": 2,
      "source": "agent",
      "created_at": "2026-04-12T12:01:00Z"
    }
  ],
  "created_at": "2026-04-12T12:00:00Z",
  "updated_at": "2026-04-12T12:01:00Z"
}
```

常见失败：
- `401 Unauthorized`
  - `unauthorized`
- `403 Forbidden`
  - `session_forbidden`
- `404 Not Found`
  - `session_not_found`

### `POST /sessions/{id}/messages`

向当前用户拥有的会话发送消息。

请求体：

```json
{
  "content": "根据规则，法师如何准备法术？"
}
```

说明：
- 前端不要再传 `user_id`
- 前端不要再传 `user_name`
- 用户身份由当前登录态决定

成功响应：
- `200 OK`

返回完整会话对象，结构与 `GET /sessions/{id}` 相同。

## 3. 常见错误码

### 认证
- `invalid_email_format`
- `invalid_email_domain`
- `password_mismatch`
- `weak_password`
- `email_already_registered`
- `invalid_credentials`
- `user_disabled`
- `unauthorized`

### 会话
- `invalid_request`
- `session_not_found`
- `session_forbidden`
- `internal_error`

## 4. 推荐前端调用流程

### 初始化
1. 调 `GET /auth/me`
2. 如果已登录，进入应用
3. 如果未登录，跳转登录页

### 注册
1. 调 `POST /auth/register`
2. 成功后直接进入应用

### 登录
1. 调 `POST /auth/login`
2. 成功后进入应用

### 多会话
1. 应用首页调用 `GET /sessions`
2. 点击某个会话，调用 `GET /sessions/{id}`
3. 新建会话时调用 `POST /sessions`

### 聊天
1. 用户输入消息
2. 调 `POST /sessions/{id}/messages`
3. 使用返回的完整 `history` 重绘消息列表
