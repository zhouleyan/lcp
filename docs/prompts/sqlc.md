# SQLC

## 前置条件
1. 所有操作仅限于当前 working 分支，不要切换其他代码分支，不要查看、引用、参考其他分支内容

## 需求
1. 基于 sqlc 实现 PG 数据库的操作
2. 在 pkg/db 下集中定义所有数据表与 sqlc 配置文件
3. 操作数据库代码通过 sqlc 生成到对应位置：按模块（例如pkg/apis/core、pkg/apis/iam），按资源类型（如pkg/apis/iam/user）