import type { Messages } from "../../types"

const common: Messages = {
  // common
  "common.name": "名称",
  "common.displayName": "显示名称",
  "common.status": "状态",
  "common.created": "创建时间",
  "common.total": "共 {count} 个",
  "common.delete": "删除",
  "common.description": "描述",
  "common.edit": "编辑",
  "common.cancel": "取消",
  "common.confirm": "确认",
  "common.save": "保存",
  "common.search": "搜索",
  "common.actions": "操作",
  "common.active": "活跃",
  "common.inactive": "停用",
  "common.all": "全部",
  "common.noPermission": "您没有权限访问此页面",
  "common.phone": "手机号",
  "common.password": "密码",
  "common.previous": "上一页",
  "common.next": "下一页",
  "common.pageSize": "每页",
  "common.page": "第 {page} 页，共 {total} 页",
  "common.noSearchResults": "未找到匹配结果",
  "common.reset": "重置",
  "common.updated": "更新时间",

  // auth
  "auth.authenticating": "登录中...",
  "auth.missingCode": "缺少授权码",

  // login
  "login.title": "LCP Console",
  "login.username": "用户名",
  "login.password": "密码",
  "login.usernamePlaceholder": "请输入用户名",
  "login.passwordPlaceholder": "请输入密码",
  "login.signIn": "登录",

  // nav
  "nav.overview": "概览",
  "nav.dashboard": "仪表盘",
  "nav.iam": "组织",
  "nav.workspaces": "租户",
  "nav.namespaces": "项目",
  "nav.users": "用户",
  "nav.roles": "角色管理",
  "nav.audit": "审计",
  "nav.auditLogs": "审计日志",
  "nav.rolebindings": "角色绑定",
  "nav.infra": "基础设施",
  "nav.hosts": "主机",
  "nav.environments": "环境",
  "nav.regions": "区域",
  "nav.sites": "站点",
  "nav.locations": "机房",
  "nav.apiDocs": "API 文档",

  // overview
  "overview.platform.title": "平台概览",
  "overview.platform.desc": "查看平台整体资源概况",
  "overview.workspace.title": "租户概览",
  "overview.workspace.desc": "查看当前租户的资源概况",
  "overview.namespace.title": "项目概览",
  "overview.namespace.desc": "查看当前项目的资源概况",
  "overview.forbidden": "您没有查看概览统计的权限，请通过侧边栏导航到您有权限的页面。",

  // scope selector
  "scope.allWorkspaces": "所有租户",
  "scope.allNamespaces": "所有项目",
  "scope.selectWorkspace": "选择租户",
  "scope.selectNamespace": "选择项目",

  // permission verb wildcards
  "perm.group.all": "全部权限",
  "perm.verb.list": "所有列表 (*:list)",
  "perm.verb.get": "所有详情 (*:get)",
  "perm.verb.create": "所有创建 (*:create)",
  "perm.verb.update": "所有更新 (*:update)",
  "perm.verb.patch": "所有修改 (*:patch)",
  "perm.verb.delete": "所有删除 (*:delete)",
  "perm.verb.deleteCollection": "所有批量删除 (*:deleteCollection)",

  // permission verb groups
  "perm.verbGroup.read": "查询",
  "perm.verbGroup.create": "创建",
  "perm.verbGroup.update": "更新",
  "perm.verbGroup.delete": "删除",

  // error
  "error.400.title": "请求错误",
  "error.400.desc": "请求无法处理，请重试。",
  "error.401.title": "未授权",
  "error.401.desc": "请登录后继续。",
  "error.403.title": "禁止访问",
  "error.403.desc": "您没有权限访问此页面。",
  "error.404.title": "页面不存在",
  "error.404.desc": "您访问的页面不存在。",
  "error.500.title": "服务器错误",
  "error.500.desc": "系统出现问题，请稍后再试。",
  "error.backHome": "返回首页",
  "error.switchAccount": "切换账号",

  // login errors
  "login.error.invalidCredentials": "用户名或密码错误",
  "login.error.accountInactive": "账号已被停用",
  "login.error.sessionExpired": "登录会话已过期，正在重新跳转...",
  "login.error.failed": "登录失败，请重试",

  // api errors
  "api.error.badRequest": "请求参数错误",
  "api.error.notFound": "{resource}不存在",
  "api.error.conflict": "{resource}已存在",
  "api.error.memberLimitExceeded": "项目成员数已达上限",
  "api.error.cannotDeleteWorkspace": "无法删除租户：仍包含项目，请先删除所有项目",
  "api.error.cannotDeleteNamespace": "无法删除项目：仍包含成员，请先移除所有成员",
  "api.error.cannotRemoveOwner": "无法移除所有者",
  "api.error.oldPasswordIncorrect": "当前密码不正确",
  "api.error.forbidden": "您没有权限执行此操作",
  "api.error.internalError": "服务器内部错误，请稍后重试",

  // validation errors
  "api.validation.required": "{field}不能为空",
  "api.validation.username.format": "用户名需为3-50位字母、数字或下划线",
  "api.validation.email.format": "请输入有效的邮箱地址",
  "api.validation.phone.format": "请输入有效的手机号（如 13800138000）",
  "api.validation.password.length": "密码长度需为8-128位",
  "api.validation.password.uppercase": "密码需包含至少一个大写字母",
  "api.validation.password.lowercase": "密码需包含至少一个小写字母",
  "api.validation.password.digit": "密码需包含至少一个数字",
  "api.validation.name.format": "名称需为3-50位小写字母、数字或连字符",
  "api.validation.rackCapacity.min": "机柜容量不能小于 0",
  "api.validation.status.format": "状态必须为「活跃」或「停用」",
  "api.validation.username.taken": "该用户名已被使用",
  "api.validation.email.taken": "该邮箱已被使用",
  "api.validation.phone.taken": "该手机号已被使用",
  "api.validation.password.hint": "8-128位，需包含大写字母、小写字母和数字",

  // action feedback
  "action.createSuccess": "创建成功",
  "action.updateSuccess": "更新成功",
  "action.deleteSuccess": "删除成功",
  "action.changePasswordSuccess": "密码修改成功",

  // user menu
  "userMenu.profile": "个人信息",
  "userMenu.changePassword": "修改密码",
  "userMenu.logout": "退出登录",
  "userMenu.oldPassword": "当前密码",
  "userMenu.newPassword": "新密码",
  "userMenu.confirmPassword": "确认新密码",
  "userMenu.passwordMismatch": "两次输入的密码不一致",
  "userMenu.passwordSameAsOld": "新密码不能与当前密码相同",
}

export default common
