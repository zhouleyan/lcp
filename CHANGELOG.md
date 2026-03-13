# Changelog

## 2026-03-13

### Added

- **O11Y 模块** — 新增监控端点（Endpoints）完整功能，包括后端 API（迁移、存储、REST、验证）、前端页面（列表、创建/编辑/删除）、OpenAPI 文档生成、国际化翻译
- **OverviewCard 共享组件** — 提取统计卡片为公共组件，供 Dashboard 和各详情页复用
- **Host IP 分配用户故事文档**

### Changed

- **数据库迁移版本号** — 从 6 位顺序编号切换为 14 位 UTC 时间戳格式（`YYYYMMDDHHmmss`）
- **前端布局样式统一化** — 统一表格边框、工具栏间距、Card 间距（`space-y-6`）、Dialog 滚动模式（flex 布局固定 footer）；详情页 stat card grid 增加响应式断点；移除不一致的返回按钮，统一使用 ConfirmDialog 组件

### Fixed

- **侧边栏溢出** — 使用 `fixed inset-0` 定位防止菜单内容过多时页面整体滚动
- **Scope 选择器导航错误** — 在仅限平台级的页面（如监控端点、网络）切换工作空间时，现在正确跳转到目标 scope 下的第一个有权限页面，而非重定向到平台首页
