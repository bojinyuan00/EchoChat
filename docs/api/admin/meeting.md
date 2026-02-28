# 管理端 — 会议管理 API

> 通用规范（认证方式、响应格式、错误码）见 [README.md](../README.md)
> 以下所有接口均需要 **JWT 认证 + admin/super_admin 角色**

---

## 接口列表

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /api/v1/admin/meetings | admin | 获取会议列表 |
| GET | /api/v1/admin/meetings/:id | admin | 获取会议详情 |
| PUT | /api/v1/admin/meetings/:id/close | admin | 强制结束会议 |
| GET | /api/v1/admin/meetings/stats | admin | 获取会议统计 |

---

## 1. 获取会议列表

`GET /api/v1/admin/meetings`

**权限：** admin

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| status | int | 无 | 按状态筛选：0=未开始，1=进行中，2=已结束 |
| type | int | 无 | 按类型筛选：1=即时会议，2=预约会议 |
| keyword | string | 无 | 搜索关键词（匹配会议标题） |
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页数量 |

---

## 2. 获取会议详情

`GET /api/v1/admin/meetings/:id`

**权限：** admin

**说明：** 返回会议完整信息，包括参与者列表、会议设置、时长统计等。

---

## 3. 强制结束会议

`PUT /api/v1/admin/meetings/:id/close`

**权限：** admin

**说明：** 强制结束后，所有参会者将收到会议结束通知，所有媒体资源将被回收。操作将记录到管理日志（`admin_operation_logs` 表）。

---

## 4. 获取会议统计

`GET /api/v1/admin/meetings/stats`

**权限：** admin

**成功响应：**
```json
{
    "code": 0,
    "message": "ok",
    "data": {
        "total_meetings": 1500,
        "ongoing_meetings": 5,
        "today_meetings": 23,
        "total_participants": 8500,
        "avg_duration": 1800
    }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| total_meetings | int | 历史会议总数 |
| ongoing_meetings | int | 当前进行中的会议数 |
| today_meetings | int | 今日创建的会议数 |
| total_participants | int | 历史累计参会人次 |
| avg_duration | int | 平均会议时长（秒） |
