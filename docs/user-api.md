# 用户接口 API 文档

本文档详细说明了 Forxi-Go 认证系统的用户相关 API 接口。

## 目录

- [通用说明](#通用说明)
- [认证说明](#认证说明)
- [用户接口](#用户接口)
- [认证接口](#认证接口)
- [OAuth接口](#oauth接口)
- [文件预览接口](#文件预览接口)

---

## 通用说明

### 基础URL

```
开发环境: http://localhost:8080
生产环境: https://forxi.cn
```

### 响应格式

所有接口返回统一的响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 状态码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |

---

## 认证说明

### JWT令牌

需要认证的接口需要在请求头中携带JWT令牌：

```
Authorization: Bearer <your-jwt-token>
```

---

## 用户接口

### 发送验证码

**接口**: `POST /api/users/send-code`

**描述**: 向指定邮箱发送注册验证码。

**角色**: 无需登录 (匿名)

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 邮箱地址 |

---

### 用户注册

**接口**: `POST /api/users/register`

**描述**: 使用邮箱、密码和验证码注册新用户。

**角色**: 无需登录 (匿名)

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码 |
| nickname | string | 是 | 昵称 |
| verification_code | string | 是 | 6位验证码 |

---

### 获取用户信息

**接口**: `GET /api/users/profile`

**描述**: 获取当前登录用户的详细信息。

**角色**: 需要登录

**认证**: 需要

---

### 更新用户资料

**接口**: `PUT /api/users/profile`

**描述**: 更新当前登录用户的资料信息。

**角色**: 需要登录

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| nickname | string | 否 | 昵称 |
| avatar | string | 否 | 头像URL |
| bio | string | 否 | 个人简介 |

---

### 修改密码

**接口**: `PUT /api/users/password`

**描述**: 修改当前登录用户的密码。

**角色**: 需要登录

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| old_password | string | 是 | 当前密码 |
| new_password | string | 是 | 新密码 |

#### 请求示例

```http
PUT /api/users/password HTTP/1.1
Authorization: Bearer <token>
Content-Type: application/json

{
    "old_password": "old_password",
    "new_password": "new_password"
}
```

#### 响应示例

```json
{
    "code": 200,
    "message": "密码修改成功",
    "data": null
}
```

### 上传文件

#### 请求示例

```http
POST /api/upload HTTP/1.1
Content-Type: multipart/form-data

scene: avatar
file: [file]
```

#### 响应示例

```json
{
    "code": 200,
    "message": "success",
    "data": {
        "url": "https://cdn.forxi.cn/avatar/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11.jpg"
    }
}
```

#### 说明

- 路径：`POST /api/upload`
- 参数：
  - `scene`: 场景类型，如 `avatar`
  - `file`: 要上传的文件
- 文件名使用 UUID 生成

#### 场景配置

| 场景 | 允许类型 | 最大大小 | 需要登录 |
|------|----------|----------|----------|
| avatar | image/jpeg, image/png, image/gif, image/webp | 2MB | 是 |

#### 更新用户头像

1. 调用上传接口上传头像文件，获取 URL
2. 在更新用户资料接口（`PUT /api/users/profile`）的 `avatar` 字段传入获得的 URL

---

## 认证接口

### 用户登录

**接口**: `POST /api/auth/login`

**描述**: 使用邮箱和密码登录。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 邮箱地址 |
| password | string | 是 | 密码 |

**成功响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "xxx",
    "refresh_token": "xxx",
    "expires_in": 86400,
    "token_type": "Bearer"
  }
}
```

---

### 刷新令牌

**接口**: `POST /api/auth/refresh`

**描述**: 使用刷新令牌获取新的访问令牌。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| refresh_token | string | 是 | 刷新令牌 |

---

### 用户登出

**接口**: `POST /api/auth/logout`

**描述**: 用户登出。

**角色**: 需要登录

**认证**: 需要

---

### 请求密码重置

**接口**: `POST /api/auth/password/reset`

**描述**: 请求密码重置。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| email | string | 是 | 邮箱地址 |

---

### 确认密码重置

**接口**: `POST /api/auth/password/reset/confirm`

**描述**: 使用重置令牌重置密码。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| token | string | 是 | 重置令牌 |
| password | string | 是 | 新密码 |

---

### 获取登录日志

**接口**: `GET /api/auth/login-logs`

**描述**: 获取当前用户的登录日志。

**角色**: 需要登录

**认证**: 需要

**查询参数**:

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| page | int | 1 | 页码 |
| pageSize | int | 10 | 每页数量 |

---

## OAuth接口

### GitHub授权

**接口**: `GET /api/oauth/github/authorize`

**描述**: 获取GitHub授权URL。

**角色**: 无需登录

**可选查询参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| bind_token | string | 已登录用户绑定OAuth时使用 |

---

### GitHub回调

**接口**: `GET /api/oauth/github/callback`

**描述**: 处理GitHub授权回调。

**角色**: 无需登录

**查询参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| code | string | 是 | 授权码 |
| state | string | 否 | 状态参数 |

**返回情况**:

1. **已绑定** - 直接登录成功，重定向到：
   ```
   {frontend_callback_url}?access_token=xxx&refresh_token=xxx&user_id=xxx
   ```

2. **需要绑定邮箱**（GitHub未返回邮箱），重定向到：
   ```
   {frontend_callback_url}?needs_email_bind=true&bind_token=xxx
   ```

---

### GitHub绑定邮箱

**接口**: `POST /api/oauth/github/bind-email`

**描述**: 首次GitHub登录时（GitHub未返回邮箱），绑定邮箱后完成注册并登录。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| bind_token | string | 是 | 绑定令牌（从回调获取） |
| email | string | 是 | 要绑定的邮箱地址 |
| email_code | string | 是 | 邮箱验证码 |
| password | string | 是 | 登录密码（至少8位） |
| confirm_password | string | 是 | 确认密码 |
| nickname | string | 否 | 昵称（可选） |

**返回情况**:

- **新邮箱**: 创建用户 + 绑定OAuth → 返回登录信息
- **已存在邮箱**: 验证密码 → 绑定OAuth → 返回登录信息

---

### 解绑第三方账号

**接口**: `POST /api/oauth/unbind`

**描述**: 解绑当前用户的第三方账号。

**角色**: 需要登录

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| provider | string | 是 | github |

---

### 获取OAuth账号列表

**接口**: `GET /api/oauth/accounts`

**描述**: 获取当前用户的OAuth账号列表。

**角色**: 需要登录

**认证**: 需要

**成功响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "provider": "github",
      "provider_user_id": "xxx",
      "bound_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

## 文件预览接口

### 在线预览

**接口**: `GET /api/filereview/online`

**描述**: 通过URL在线预览文件，服务器会下载文件并进行预览。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| url | string | 是 | 文件URL地址 |

**支持的文件类型**:

- 文本: txt, md, json, log, xml, html, htm, css, js, ts, py, java, go, rs, c, cpp, h, hpp, cs, php, rb, sh, sql, yaml, yml, toml, ini, conf, cfg, env
- 图片: jpg, jpeg, png, gif, webp, svg, ico, bmp
- 视频: mp4, webm, avi, mov
- 文档: pdf
- Office: doc, docx, xls, xlsx, ppt, pptx

**限制**:
- 文件大小不能超过配置的最大值（默认 5MB）
- 仅支持 http:// 和 https:// 协议
- 不支持本地或内网地址

---

### 本地预览

**接口**: `POST /api/filereview/local`

**描述**: 上传本地文件进行预览。

**角色**: 无需登录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | 要上传的文件 |

**支持的文件类型**: 同在线预览

**限制**: 文件大小不能超过配置的最大值（默认 5MB）

**返回格式**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "type": "image|pdf|video|office|text",
    "content": "base64编码的内容（图片、视频、pdf、文本时返回）",
    "file": "文件名（office文件时返回，用于获取预览链接）",
    "name": "原始文件名",
    "mime": "文件MIME类型"
  }
}
```

---

### 文件缓存

**接口**: `GET /api/filereview/cache`

**描述**: 访问office文件的预览/下载链接。

**角色**: 无需登录

**查询参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | string | 是 | 文件名（local接口返回的file字段） |

**返回**: 直接返回文件内容

**注意**: 链接有效期为配置的文件缓存时间（默认30分钟）

---

## 错误码说明

### 业务错误码

| 错误码 | 说明 |
|--------|------|
| 1001 | 邮箱已存在 |
| 1002 | 用户不存在 |
| 1003 | 密码错误 |
| 1004 | 令牌无效或过期 |
| 1005 | 账号被禁用 |
| 1006 | 邮箱未验证 |
| 1007 | OAuth账号已绑定 |
| 1008 | OAuth账号不存在 |
