# 管理端 — 系统管理 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 以下所有接口均需要 **JWT 认证 + admin/super_admin 角色**

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/dashboard | admin | 获取仪表盘数据 |
| GET | /api/v1/admin/logs | admin | 获取操作日志 |
| GET | /api/v1/admin/system/config | super_admin | 获取系统配置 |
| PUT | /api/v1/admin/system/config | super_admin | 更新系统配置 |

---

## 1. 获取仪表盘数据

`GET /api/v1/admin/dashboard`

**权限：** admin

**说明：** 返回系统核心指标概览，用于后台管理端首页展示。

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "total_users": 1200,
        "online_users": 85,
        "today_new_users": 12,
        "ongoing_meetings": 5,
        "today_meetings": 23,
        "today_messages": 5600
    }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| total_users | int | 系统注册用户总数 |
| online_users | int | 当前在线用户数（WebSocket 连接中） |
| today_new_users | int | 今日新注册用户数 |
| ongoing_meetings | int | 当前进行中的会议数 |
| today_meetings | int | 今日创建的会议数 |
| today_messages | int | 今日消息发送总数 |

---

## 2. 获取操作日志

`GET /api/v1/admin/logs`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| module | string | 无 | 按模块筛选：user / meeting / permission / system |
| action | string | 无 | 按操作类型筛选：create / update / delete / disable / enable / close |
| admin_id | int | 无 | 按操作管理员 ID 筛选 |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

**说明：** 日志数据来源于 `admin_operation_logs` 表，记录所有管理员的操作行为，用于审计和问题追踪。

---

## 3. 获取系统配置

`GET /api/v1/admin/system/config`

**权限：** super_admin

**说明：** 返回系统全局配置项，如默认会议最大人数、消息保留天数等。

---

## 4. 更新系统配置

`PUT /api/v1/admin/system/config`

**权限：** super_admin

**说明：** 修改系统全局配置，操作将记录到管理日志。
