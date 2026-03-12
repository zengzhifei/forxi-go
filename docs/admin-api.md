# 管理员接口 API 文档

本文档详细说明了 Forxi-Go 认证系统的管理员相关 API 接口。

## 目录

- [通用说明](#通用说明)
- [角色说明](#角色说明)
- [认证说明](#认证说明)
- [管理员接口](#管理员接口)
- [超级管理员接口](#超级管理员接口)

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

## 角色说明

系统中有三种用户角色：

| 角色 | 说明 | 权限 |
|------|------|------|
| `user` | 普通用户 | 查看个人信息、修改个人资料、绑定/解绑第三方账号 |
| `admin` | 管理员 | 普通用户权限 + 查看用户列表、修改用户状态 |
| `super_admin` | 超级管理员 | 管理员权限 + 修改用户角色 |

---

## 认证说明

### JWT令牌

需要认证的接口需要在请求头中携带JWT令牌：

```
Authorization: Bearer <your-jwt-token>
```

### 权限要求

- `/admin/*` 接口需要 `admin` 或 `super_admin` 角色
- 部分接口仅限 `super_admin` 访问

---

## 管理员接口

以下接口需要 `admin` 或 `super_admin` 角色访问。

### 获取用户列表

**接口**: `GET /admin/users`

**描述**: 获取系统中的用户列表。

**角色**: 需要 `admin` 或 `super_admin`

**认证**: 需要

**查询参数**:

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| page | int | 1 | 页码 |
| pageSize | int | 10 | 每页数量（最大100） |

**成功响应** (200):

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "user_id": 1234567890,
      "email": "user1@example.com",
      "nickname": "User1",
      "avatar": "",
      "bio": "",
      "role": "user",
      "email_verified": true,
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    {
      "user_id": 1234567891,
      "email": "admin@example.com",
      "nickname": "Admin",
      "avatar": "",
      "bio": "",
      "role": "admin",
      "email_verified": true,
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "pageSize": 10,
    "total": 100,
    "totalPages": 10
  }
}
```

---

### 更新用户状态

**接口**: `PUT /admin/users/:id/status`

**描述**: 更新指定用户的状态。可以将用户设为 active（激活）、inactive（未激活）或 banned（禁用）。

**角色**: 需要 `admin` 或 `super_admin`

**认证**: 需要

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | int | 用户的user_id |

**请求参数**:

```json
{
  "status": "inactive"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| status | string | 是 | 状态值：active / inactive / banned |

**成功响应** (200):

```json
{
  "code": 0,
  "message": "User status updated successfully",
  "data": null
}
```

**错误响应**:

- 用户不存在: `"用户不存在"`

---

## 超级管理员接口

以下接口仅限 `super_admin` 角色访问。

### 更新用户角色

**接口**: `PUT /admin/users/:id/role`

**描述**: 更新指定用户的角色。可以将用户设为 user（普通用户）、admin（管理员）或 super_admin（超级管理员）。

**角色**: 需要 `super_admin`

**认证**: 需要

**路径参数**:

| 参数名 | 类型 | 说明 |
|--------|------|------|
| id | int | 用户的user_id |

**请求参数**:

```json
{
  "role": "admin"
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| role | string | 是 | 角色值：user / admin / super_admin |

**成功响应** (200):

```json
{
  "code": 0,
  "message": "User role updated successfully",
  "data": null
}
```

**错误响应**:

- 用户不存在: `"用户不存在"`

---

## 使用示例

### 使用示例（Python）

```python
import requests

BASE_URL = "http://localhost:8080"
ADMIN_TOKEN = "your-admin-jwt-token"

headers = {
    "Authorization": f"Bearer {ADMIN_TOKEN}"
}

# 获取用户列表
response = requests.get(
    f"{BASE_URL}/admin/users",
    params={"page": 1, "pageSize": 10},
    headers=headers
)
print(response.json())

# 更新用户状态
response = requests.put(
    f"{BASE_URL}/admin/users/1234567890/status",
    json={"status": "banned"},
    headers=headers
)
print(response.json())

# 更新用户角色（仅super_admin）
response = requests.put(
    f"{BASE_URL}/admin/users/1234567890/role",
    json={"role": "admin"},
    headers=headers
)
print(response.json())
```

### 使用示例（cURL）

```bash
# 获取用户列表
curl -X GET "http://localhost:8080/admin/users?page=1&pageSize=10" \
  -H "Authorization: Bearer <admin-token>"

# 更新用户状态
curl -X PUT "http://localhost:8080/admin/users/1234567890/status" \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "banned"}'

# 更新用户角色（仅super_admin）
curl -X PUT "http://localhost:8080/admin/users/1234567890/role" \
  -H "Authorization: Bearer <super-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

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
