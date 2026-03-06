import type { Messages } from "../types"

const zhCN: Messages = {
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
  "common.phone": "手机号",
  "common.password": "密码",
  "common.previous": "上一页",
  "common.next": "下一页",
  "common.page": "第 {page} 页，共 {total} 页",

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
  "nav.iam": "组织",
  "nav.workspaces": "工作空间",
  "nav.namespaces": "命名空间",
  "nav.users": "用户",
  "nav.apiDocs": "API 文档",

  // workspace
  "workspace.title": "工作空间",
  "workspace.manage": "管理工作空间。共 {count} 个。",
  "workspace.create": "创建工作空间",
  "workspace.notFound": "工作空间未找到。",
  "workspace.noData": "暂无工作空间。",
  "workspace.backToList": "返回工作空间列表",
  "workspace.details": "详情",
  "workspace.overview": "概览",
  "workspace.namespaces": "命名空间",
  "workspace.members": "成员",
  "workspace.ownerId": "所有者 ID",
  "workspace.namespacesComingSoon": "命名空间列表即将推出。",
  "workspace.membersComingSoon": "成员管理即将推出。",

  // namespace
  "namespace.title": "命名空间",
  "namespace.manage": "管理命名空间。共 {count} 个。",
  "namespace.create": "创建命名空间",
  "namespace.noData": "暂无命名空间。",
  "namespace.workspaceId": "工作空间 ID",
  "namespace.visibility": "可见性",

  // user
  "user.title": "用户",
  "user.manage": "管理平台用户。共 {count} 个。",
  "user.create": "创建用户",
  "user.edit": "编辑用户",
  "user.noData": "暂无用户。",
  "user.username": "用户名",
  "user.email": "邮箱",
  "user.searchPlaceholder": "搜索用户名、邮箱、手机号、显示名称...",
  "common.updated": "更新时间",
  "user.deleteConfirm": "确定要删除用户「{name}」吗？此操作不可撤销。",
  "user.batchDelete": "批量删除",
  "user.batchDeleteConfirm": "确定要删除选中的 {count} 个用户吗？此操作不可撤销。",

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

  // login errors
  "login.error.invalidCredentials": "用户名或密码错误",
  "login.error.accountInactive": "账号已被停用",
  "login.error.failed": "登录失败，请重试",

  // api errors
  "api.error.badRequest": "请求参数错误",
  "api.error.notFound": "{resource}不存在",
  "api.error.conflict": "{resource}已存在",
  "api.error.oldPasswordIncorrect": "当前密码不正确",
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

export default zhCN
