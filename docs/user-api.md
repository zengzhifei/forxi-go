# 用户接口 API 文档

本文档详细说明了 Forxi-Go 认证系统的用户相关 API 接口。

## 目录

- [通用说明](#通用说明)
- [认证说明](#认证说明)
- [用户接口](#用户接口)
- [认证接口](#认证接口)
- [OAuth接口](#oauth接口)

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

---

## OAuth接口

### GitHub授权

**接口**: `GET /api/oauth/github/authorize`

**描述**: 获取GitHub授权URL。

**角色**: 无需登录

---

### GitHub回调

**接口**: `GET /api/oauth/github/callback`

**描述**: 处理GitHub授权回调。

**角色**: 无需登录

**查询参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| code | string | 是 | 授权码 |

**返回情况**:

1. **已绑定** - 直接登录成功：
   ```
   {frontend_callback_url}?access_token=xxx&refresh_token=xxx&user_id=xxx
   ```

2. **需要绑定邮箱**：
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

**返回情况**:

- **新邮箱**: 创建用户 + 绑定OAuth → 返回登录信息
- **已存在邮箱**: 验证密码 → 绑定OAuth → 返回登录信息

---

### 微信授权

**接口**: `GET /api/oauth/wechat/authorize`

**描述**: 获取微信授权URL。

**角色**: 无需登录

---

### 微信回调

**接口**: `GET /api/oauth/wechat/callback`

**描述**: 处理微信授权回调。

**角色**: 无需登录

---

### 绑定第三方账号

**接口**: `POST /api/oauth/bind`

**描述**: 将第三方账号绑定到当前已登录用户。

**角色**: 需要登录

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| provider | string | 是 | github/wechat |
| provider_user_id | string | 是 | 第三方用户ID |
| access_token | string | 是 | 访问令牌 |
| refresh_token | string | 否 | 刷新令牌 |

---

### 解绑第三方账号

**接口**: `POST /api/oauth/unbind`

**描述**: 解绑当前用户的第三方账号。

**角色**: 需要登录

**认证**: 需要

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| provider | string | 是 | github/wechat |

---

### 获取OAuth账号列表

**接口**: `GET /api/oauth/accounts`

**描述**: 获取当前用户的OAuth账号列表。

**角色**: 需要登录

**认证**: 需要
