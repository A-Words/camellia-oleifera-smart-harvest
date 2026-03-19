# PRD：油茶智能采摘辅助系统

- 文档版本：`v2.0`
- 文档日期：`2026-03-19`
- 仓库：`camellia-oleifera-smart-harvest`

## 1. 产品目标

本项目围绕油茶果采收场景，建设一个三层软件系统：

1. `recognition`：识别油茶果位置并输出目标检测与数量统计基线。
2. `decision`：将识别结果转化为树优先级、区域优先级、采摘顺序和跳过建议。
3. `operations`：沉淀地块、树木档案、进度、效率、预计产量与完成率。

当前阶段目标是完成新的目录与术语基线，并保持最小可运行链路：

`operator-console -> api-gateway -> recognition-api`

## 2. 范围定义

### 2.1 当前 In Scope

1. 识别层页面与服务迁移到新目录和新命名。
2. 网关仅保留系统健康和识别代理主线。
3. 新公开路由固定为 `/recognition`、`/decision`、`/operations`。
4. 新公开接口固定为：
   - `GET /healthz`
   - `GET /v1/health`
   - `POST /v1/recognition/image`
   - `GET /v1/recognition/stream`
5. `decision` 与 `operations` 提供骨架页和目录占位。

### 2.2 当前 Out of Scope

1. 旧的 `batch / trace / dashboard` 页面和接口。
2. 区块链溯源、补链、公开验真。
3. 决策层与作业层的完整后端实现。
4. 机械臂、IoT、机器人等硬件接入。

## 3. 用户与场景

### 3.1 主要用户

1. 采摘人员或现场操作员
2. 果园管理人员
3. 研发与联调团队

### 3.2 当前核心场景

1. 操作员进入 `/recognition`，选择摄像头并开始实时识别。
2. 前端通过网关 WebSocket 接收识别结果，展示当前帧目标数量、会话累计检测数，并将检测框叠加到实时画面。
3. 用户切换到 `/decision` 或 `/operations` 查看下一阶段能力骨架。

## 4. 业务域定义

### 4.1 识别结果域

识别结果域当前关注：

- 果实位置
- 单类检测标签：`camellia_oleifera_fruit`
- 当前帧目标数量
- 会话累计检测数量
- 数据集与模型产物的可追踪训练基线

### 4.2 作业决策域

作业决策域后续将负责：

- 树优先级
- 树冠区域优先级
- 采摘顺序
- 跳过建议

### 4.3 作业管理域

作业管理域后续将负责：

- 地块管理
- 树木档案
- 采摘进度
- 人员效率
- 预计产量
- 完成率

## 5. 系统结构

### 5.1 架构边界

- `clients/operator-console`：识别、决策、作业三层前端入口
- `services/api-gateway`：统一入口、鉴权、限流、代理
- `services/recognition-api`：视觉识别与识别结果输出

### 5.2 当前目录落位

- `shared/contracts/openapi.yaml`：当前对外契约
- `shared/domain/`：共享领域名词
- `tooling/config/`：配置
- `tooling/scripts/`：脚本
- `mlops/`：训练、数据、产物

## 6. 当前功能需求

### 6.1 识别页 `/recognition`

必须支持：

1. 摄像头设备枚举与切换
2. 实时流识别
3. 当前帧检测摘要与目标框叠层展示
4. 会话级检测数量汇总展示
5. 明确提示当前主线为检测基线，成熟度能力待后续数据补齐

### 6.2 决策页 `/decision`

当前允许：

1. 展示决策层能力说明
2. 保留后续实现入口

### 6.3 作业页 `/operations`

当前允许：

1. 展示作业管理层能力说明
2. 保留后续实现入口

## 7. 接口基线

### 7.1 系统接口

- `GET /healthz`
- `GET /v1/health`

### 7.2 识别接口

- `POST /v1/recognition/image`
- `GET /v1/recognition/stream`

### 7.3 契约约束

- 契约文件：`shared/contracts/openapi.yaml`
- 当前标签组：
  - `system`
  - `recognition`
  - `decision`
  - `operations`

## 8. 非功能要求

1. 前端不得绕过网关直连识别服务。
2. 目录、命令、配置路径必须与文档保持一致。
3. 识别服务在模型未成功加载时仍需暴露健康状态，用于排障。

## 9. 验收标准

1. `/recognition` 页面可打开并正常建立识别流。
2. `/recognition` 页面可在实时画面上显示油茶果检测框与置信度标签。
3. `GET /healthz` 和 `GET /v1/health` 可返回健康信息。
4. `POST /v1/recognition/image` 可返回识别结果。
5. `GET /v1/recognition/stream` 可返回 `frame` 与 `summary` 事件。
6. `/decision` 与 `/operations` 页面可正常渲染骨架。
7. 旧 `/batch/create`、`/trace/*`、`/dashboard` 不再作为主线路由和文档基线。
