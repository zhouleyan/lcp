# CMDB 前端设计：Region / Site / Location 页面

> 日期：2026-03-12
> 模块：Infra 前端扩展
> 状态：已确认

## 路由

纯平台级，无 workspace/namespace 前缀：

```
/infra/regions                    → RegionListPage
/infra/regions/:regionId          → RegionDetailPage（含 Sites 子列表）
/infra/sites                      → SiteListPage
/infra/sites/:siteId              → SiteDetailPage（含 Locations 子列表）
/infra/locations                  → LocationListPage
/infra/locations/:locationId      → LocationDetailPage
```

## 导航

Infra 组下新增三项，scope 仅 `["platform"]`：
- 区域（Regions）— MapPin 图标
- 站点（Sites）— Building2 图标
- 机房（Locations）— Warehouse 图标

## 列表页

| 资源 | 搜索 | 筛选 | 表格列 | 排序 |
|---|---|---|---|---|
| Region | name/displayName | status | 名称、显示名、状态、站点数、创建时间 | name, created_at |
| Site | name/displayName | status, regionId | 名称、显示名、所属区域、状态、机房数、创建时间 | name, created_at |
| Location | name/displayName | status, siteId, regionId | 名称、显示名、所属站点、所属区域、楼层、机柜容量、状态、创建时间 | name, created_at |

## 详情页

- Region 详情：基本信息卡片 + 下属 Sites 表格
- Site 详情：基本信息 + 联系人信息 + 下属 Locations 表格
- Location 详情：基本信息 + 联系人信息

## 表单字段

| Region | Site | Location |
|---|---|---|
| name | name | name |
| displayName | displayName | displayName |
| description | description | description |
| status | regionId (Select) | siteId (Select) |
| latitude | status | status |
| longitude | address | floor |
| | latitude | rackCapacity |
| | longitude | contactName |
| | contactName | contactPhone |
| | contactPhone | contactEmail |
| | contactEmail | |

## 文件结构

新建：
```
src/api/infra/regions.ts
src/api/infra/sites.ts
src/api/infra/locations.ts
src/pages/infra/regions/list.tsx
src/pages/infra/regions/detail.tsx
src/pages/infra/sites/list.tsx
src/pages/infra/sites/detail.tsx
src/pages/infra/locations/list.tsx
src/pages/infra/locations/detail.tsx
```

修改：
```
src/pages/infra/routes.tsx        — 添加路由
src/lib/nav-config.ts             — 添加导航项
src/api/types.ts                  — 添加 TS 类型
src/i18n/locales/zh-CN/infra.ts   — 添加中文翻译
src/i18n/locales/en-US/infra.ts   — 添加英文翻译
src/components/app-breadcrumb.tsx  — 添加 label key 映射
```

## 权限码

```
infra:regions:list / create / get / update / patch / delete / deleteCollection
infra:sites:list / create / get / update / patch / delete / deleteCollection
infra:locations:list / create / get / update / patch / delete / deleteCollection
```
