# 前后端集成开发规范

> **适用范围**：EchoChat 项目全端（Go 后端 + admin 管理端 + frontend 用户端）
> **创建日期**：2026-03-02
> **最后更新**：2026-03-02（新增前后台 Token 隔离规范）

---

## 一、API 响应格式规范

### 1.1 后端统一响应结构

所有后端 API 必须使用 `pkg/utils/response.go` 中的 `Response*` 系列函数返回响应，保证结构一致：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "trace_id": "uuid-v4",
  "time": "2026-03-02 14:00:00"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 业务状态码：0=成功，非 0=HTTP 状态码 |
| `message` | string | 人类可读的描述信息，**前端直接展示给用户** |
| `data` | any | 业务数据，失败时为 null |
| `trace_id` | string | 请求追踪 ID，用于日志排查 |
| `time` | string | 服务器响应时间 |

### 1.2 后端 Response 函数对照

| 函数 | HTTP Status | code | 使用场景 |
|------|-------------|------|----------|
| `ResponseOK` | 200 | 0 | 请求成功 |
| `ResponseCreated` | 201 | 0 | 资源创建成功 |
| `ResponseBadRequest` | 400 | 400 | 参数错误、业务规则校验失败 |
| `ResponseUnauthorized` | 401 | 401 | 认证失败（密码错误、Token 无效/过期） |
| `ResponseForbidden` | 403 | 403 | 权限不足、账号被禁用 |
| `ResponseNotFound` | 404 | 404 | 资源不存在 |
| `ResponseError` | 500 | 500 | 服务器内部错误 |

### 1.3 message 字段要求

- **必须是中文**，面向最终用户
- **必须精准描述错误原因**，不能笼统（如 "操作失败"）
- **安全场景除外**：登录时"用户不存在"和"密码错误"统一为 "账号或密码错误"

---

## 二、HTTP 状态码使用规范

### 2.1 状态码语义

| 状态码 | 语义 | 后端使用场景 |
|--------|------|-------------|
| 200 | OK | 查询成功、操作成功 |
| 201 | Created | 资源创建成功（注册、新建用户） |
| 400 | Bad Request | 参数校验失败、业务规则不满足 |
| 401 | Unauthorized | 认证失败：密码错误、Token 过期/无效/已注销 |
| 403 | Forbidden | 权限不足：角色不匹配、账号被禁用/注销 |
| 404 | Not Found | 请求的资源不存在 |
| 500 | Internal Server Error | 服务器内部错误 |

### 2.2 安全约束

**登录接口的 401 响应**：

```
✅ 正确：用户不存在 → 401 "账号或密码错误"
✅ 正确：密码错误 → 401 "账号或密码错误"
❌ 错误：用户不存在 → 404 "用户不存在"（暴露用户是否注册）
```

防止用户枚举攻击，登录场景下所有认证失败统一返回相同状态码和信息。

---

## 三、前端错误处理规范

### 3.1 核心原则

1. **后端 message 优先**：所有 HTTP 错误提示必须优先使用 `data.message`
2. **禁止硬编码覆盖**：不能用前端硬编码的文案替换后端返回的错误信息
3. **Fallback 仅兜底**：`|| '默认文案'` 仅在后端无响应体或无 message 字段时触发
4. **按状态码分类处理**：不同 HTTP 状态码执行不同的副作用（跳转、清 Token 等）

### 3.2 管理端（admin/）错误处理

文件：`admin/src/utils/request.js`

```
HTTP 响应成功（2xx）:
  ├─ code === 0 → 返回数据
  └─ code !== 0 → ElMessage.error(data.message || '请求失败')

HTTP 响应错误：
  ├─ 401
  │   ├─ 在登录页 → ElMessage.error(data.message || '登录已过期')
  │   └─ 非登录页 → ElMessage.error(data.message || '登录已过期') + 清 Token + 跳转 /login
  ├─ 403 → ElMessage.error(data.message || '没有访问权限')
  ├─ 其他 → ElMessage.error(data.message || '请求错误(N)')
  └─ 网络异常 → ElMessage.error('网络异常，请检查网络连接')
```

**401 场景区分的关键**：通过 `router.currentRoute.value.path === '/login'` 判断当前是否在登录页。登录页不清 Token 不跳转，仅显示错误。

### 3.3 前台用户端（frontend/）错误处理

文件：`frontend/src/utils/request.js`

```
HTTP 响应成功（2xx）:
  ├─ code === 0 → resolve(data)
  └─ code !== 0 → showToast(data.message || '请求失败')

HTTP 响应错误：
  ├─ 401
  │   ├─ 登录/注册请求 → showToast(data.message || '登录已过期')
  │   └─ 其他请求 → showToast(data.message || '登录已过期') + removeToken() + 跳转登录页
  ├─ 403 → showToast(data.message || '没有访问权限')
  ├─ 其他 → showToast(data.message || '请求错误(N)')
  └─ 网络异常 → showToast('网络异常，请检查网络连接')
```

**401 场景区分的关键**：通过正则 `/\/auth\/(login|register)$/` 匹配请求 URL。登录/注册请求不清 Token 不跳转。

### 3.4 错误处理检查清单

新增 API 接口时，必须确认以下各项：

- [ ] 后端 Controller 使用正确的 `Response*` 函数
- [ ] 后端 Controller 的 error switch 覆盖所有已知业务错误
- [ ] 后端不使用 `_` 忽略 error，至少 log warning
- [ ] 前端调用方的 catch 不自行覆盖 message（交给拦截器统一处理）
- [ ] 前端页面级 catch 只做状态清理（如 loading = false），不重复弹提示

---

## 四、后端错误处理规范

### 4.1 Controller 层

- 使用 switch/case 或 if/else 匹配所有已知业务错误
- 每种错误映射到正确的 HTTP 状态码
- message 参数必须是用户友好的中文描述
- default 分支处理未知错误，返回 500

```go
if err := svc.DoSomething(ctx, req); err != nil {
    switch err {
    case service.ErrNotFound:
        utils.ResponseNotFound(c, "资源不存在")
    case service.ErrInvalidParam:
        utils.ResponseBadRequest(c, "参数无效")
    default:
        logs.Error(ctx, funcName, "操作失败", zap.Error(err))
        utils.ResponseError(c, "操作失败")
    }
    return
}
```

### 4.2 Service 层

- 定义明确的业务错误变量（`var ErrXxx = errors.New("...")`）
- 不在 Service 层直接返回 HTTP 状态码，由 Controller 映射
- DAO 层的 `gorm.ErrRecordNotFound` 必须转换为业务错误

### 4.3 禁止忽略错误

```go
// ❌ 错误：忽略 error
roles, _ := s.roleDAO.GetUserRoleCodes(ctx, user.ID)

// ✅ 正确：处理 error
roles, err := s.roleDAO.GetUserRoleCodes(ctx, user.ID)
if err != nil {
    logs.Warn(ctx, funcName, "获取角色失败", zap.Error(err))
    roles = []string{}
}
```

---

## 五、前后端错误码对照表

| 后端场景 | HTTP | message 示例 | 前端行为 |
|---------|------|-------------|---------|
| 登录-密码错误 | 401 | 账号或密码错误 | 显示 message |
| 登录-用户不存在 | 401 | 账号或密码错误 | 显示 message |
| 登录-账号被禁用 | 403 | 账号已被禁用 | 显示 message |
| Token 过期 | 401 | 认证已过期或无效，请重新登录 | 显示 message + 清 Token + 跳转 |
| Token 已注销 | 401 | 认证已失效，请重新登录 | 显示 message + 清 Token + 跳转 |
| 缺少认证信息 | 401 | 缺少认证信息 | 显示 message + 清 Token + 跳转 |
| 权限不足 | 403 | 权限不足，需要角色: admin 或 super_admin | 显示 message |
| 参数校验失败 | 400 | 参数校验失败: ... | 显示 message |
| 用户名已注册 | 400 | 用户名或邮箱已被注册 | 显示 message |
| 不能禁用自己 | 400 | 不能禁用自己的账号 | 显示 message |
| 用户不存在 | 404 | 用户不存在 | 显示 message |
| 服务器内部错误 | 500 | 获取用户列表失败 | 显示 message |
| 网络异常 | - | （无响应体） | 显示前端 fallback 文案 |

---

## 六、前后台 Token 隔离规范

### 6.1 核心原则

前台用户端（frontend）和后台管理端（admin）的 Token **必须在 Redis 中完全隔离**，同一用户可以同时在两端保持登录状态，互不影响。

### 6.2 Redis Key 格式

```
echo:auth:token:{client_type}:{user_id}     → Access Token
echo:auth:refresh:{client_type}:{user_id}    → Refresh Token
```

| client_type | 说明 | 示例 Key |
|------------|------|----------|
| `frontend` | 前台用户端 | `echo:auth:token:frontend:1` |
| `admin` | 后台管理端 | `echo:auth:token:admin:1` |

### 6.3 JWT Claims 中的 client_type

JWT Token 的 Claims 中包含 `client_type` 字段，用于：
- 中间件校验时定位正确的 Redis key
- 登出时只删除当前端的 Token
- Refresh Token 刷新时保持 client_type 不变

```json
{
  "user_id": 1,
  "username": "admin_test",
  "roles": ["user", "admin"],
  "client_type": "admin",
  "sub": "access",
  "exp": 1709400000,
  "iss": "echochat"
}
```

### 6.4 API 路由区分

| 端 | 登录 API | 说明 |
|----|---------|------|
| 前台 | `POST /api/v1/auth/login` | clientType=frontend |
| 管理端 | `POST /api/v1/admin/auth/login` | clientType=admin，额外检查管理员角色 |

### 6.5 登出行为

- 前台登出：只删除 `echo:auth:token:frontend:{user_id}`，不影响管理端
- 管理端登出：只删除 `echo:auth:token:admin:{user_id}`，不影响前台
- 管理端禁用用户：应删除该用户两端的所有 Token（由管理模块负责）

### 6.6 检查清单

新增认证相关功能时，必须确认：

- [ ] Login 方法传递了正确的 clientType
- [ ] 前端调用了正确的登录 API 端点
- [ ] Redis key 包含 clientType 前缀
- [ ] JWT Claims 中包含 client_type 字段
- [ ] 登出时只删除对应 clientType 的 Token
- [ ] Token 刷新时保持原 clientType 不变

---

## 7. 角色等级与权限管控规范

### 7.1 角色等级设计

`auth_roles` 表 `level` 字段，值越小权限越高，预留间隔：

| 角色 | Code | Level | 说明 |
|------|------|-------|------|
| 超级管理员 | super_admin | 1 | 最高权限 |
| 管理员 | admin | 10 | 后台管理 |
| 普通用户 | user | 100 | 基础权限 |

用户可拥有多个角色，取**最小 level 值**作为有效权限等级。

### 7.2 权限管控规则

**核心原则：操作者的 level 必须严格小于目标用户的 level，才能执行管理操作。**

1. **更新用户状态（禁用/启用）**：操作者 level < 目标用户 level
2. **设置用户角色**：操作者 level < 目标用户 level，且不能分配 level <= 自身的角色
3. **前端管控**：禁用/启用按钮对高等级用户隐藏，高等级角色 checkbox 禁用

### 7.3 API 响应变更

`AdminUserInfo` 的 `roles` 字段从 `[]string` 改为 `[]RoleInfo`：

```json
{
    "roles": [
        { "code": "admin", "name": "管理员", "level": 10 },
        { "code": "user", "name": "普通用户", "level": 100 }
    ],
    "max_level": 10
}
```

### 7.4 检查清单

涉及用户管理操作时，必须确认：

- [ ] 后端 Service 层调用 `checkPermissionLevel` 进行等级校验
- [ ] 权限不足时返回 403（`ErrInsufficientPermission`）
- [ ] 前端通过比较 `adminMaxLevel` 和 `targetMaxLevel` 控制 UI 可见性
- [ ] 角色分配使用全量覆盖模式（`SetUserRoles`），非追加模式
