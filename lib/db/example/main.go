package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
)

func main() {
	ctx := context.Background()

	// ============================================================
	// 1. 创建连接池
	// ============================================================
	pool, err := db.NewPool(ctx, db.Config{
		Host:     envOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     envOrDefault("DB_USER", "postgres"),
		Password: envOrDefault("DB_PASSWORD", "postgres"),
		DBName:   envOrDefault("DB_NAME", "lcp"),
		SSLMode:  "disable",
		MaxConns: 10,
	})
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	// 用 pool 初始化 sqlc 生成的 Queries
	queries := generated.New(pool)

	// ============================================================
	// 2. 创建用户
	// ============================================================
	user1, err := queries.CreateUser(ctx, generated.CreateUserParams{
		Username:    "alice",
		Email:       "alice@example.com",
		DisplayName: "Alice Wang",
		Phone:       "13800000001",
		AvatarUrl:   "https://example.com/avatars/alice.png",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create user1: %v", err)
	}
	fmt.Printf("Created user: id=%d, username=%s\n", user1.ID, user1.Username)
	user2, err := queries.CreateUser(ctx, generated.CreateUserParams{
		Username:    "bob",
		Email:       "bob@example.com",
		DisplayName: "Bob Li",
		Phone:       "13800000002",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create user2: %v", err)
	}
	fmt.Printf("Created user: id=%d, username=%s\n", user2.ID, user2.Username)

	// ============================================================
	// 3. 查询用户（按 ID / Username / Email）
	// ============================================================
	got, err := queries.GetUserByID(ctx, user1.ID)
	if err != nil {
		log.Fatalf("get user by id: %v", err)
	}
	fmt.Printf("GetUserByID: %s <%s>\n", got.Username, got.Email)

	got, err = queries.GetUserByUsername(ctx, "bob")
	if err != nil {
		log.Fatalf("get user by username: %v", err)
	}
	fmt.Printf("GetUserByUsername: %s <%s>\n", got.Username, got.Email)

	// ============================================================
	// 4. 更新用户
	// ============================================================
	updated, err := queries.UpdateUser(ctx, generated.UpdateUserParams{
		ID:          user1.ID,
		Username:    user1.Username,
		Email:       user1.Email,
		DisplayName: "Alice W.",
		Phone:       user1.Phone,
		AvatarUrl:   user1.AvatarUrl,
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("update user: %v", err)
	}
	fmt.Printf("Updated user: display_name=%s\n", updated.DisplayName)

	// 更新最后登录时间
	if err := queries.UpdateUserLastLogin(ctx, user1.ID); err != nil {
		log.Fatalf("update last login: %v", err)
	}

	// ============================================================
	// 5. 创建 Namespace
	// ============================================================
	ns1, err := queries.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        "team-alpha",
		DisplayName: "Team Alpha",
		Description: "The alpha team namespace",
		OwnerID:     user1.ID,
		Visibility:  "private",
		MaxMembers:  50,
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create namespace1: %v", err)
	}
	fmt.Printf("Created namespace: id=%d, name=%s\n", ns1.ID, ns1.Name)

	ns2, err := queries.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        "team-beta",
		DisplayName: "Team Beta",
		Description: "The beta team namespace",
		OwnerID:     user2.ID,
		Visibility:  "internal",
		MaxMembers:  100,
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create namespace2: %v", err)
	}
	fmt.Printf("Created namespace: id=%d, name=%s\n", ns2.ID, ns2.Name)
	// ============================================================
	// 6. 多对多关联：将用户加入 Namespace
	// ============================================================
	_, err = queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      user1.ID,
		NamespaceID: ns1.ID,
		Role:        "owner",
	})
	if err != nil {
		log.Fatalf("add user1 to ns1: %v", err)
	}

	_, err = queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      user2.ID,
		NamespaceID: ns1.ID,
		Role:        "member",
	})
	if err != nil {
		log.Fatalf("add user2 to ns1: %v", err)
	}

	_, err = queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      user1.ID,
		NamespaceID: ns2.ID,
		Role:        "admin",
	})
	if err != nil {
		log.Fatalf("add user1 to ns2: %v", err)
	}
	fmt.Println("Users added to namespaces")

	// ============================================================
	// 7. 关联查询：查询用户所属的 Namespace
	// ============================================================
	userNsList, err := queries.ListNamespacesByUserID(ctx, user1.ID)
	if err != nil {
		log.Fatalf("list namespaces by user: %v", err)
	}
	fmt.Printf("\nUser %s belongs to %d namespaces:\n", user1.Username, len(userNsList))
	for _, ns := range userNsList {
		fmt.Printf("  - %s (role: %s)\n", ns.Name, ns.Role)
	}

	// ============================================================
	// 8. 关联查询：查询 Namespace 下的用户
	// ============================================================
	nsUserList, err := queries.ListUsersByNamespaceID(ctx, ns1.ID)
	if err != nil {
		log.Fatalf("list users by namespace: %v", err)
	}
	fmt.Printf("\nNamespace %s has %d members:\n", ns1.Name, len(nsUserList))
	for _, u := range nsUserList {
		fmt.Printf("  - %s (role: %s)\n", u.Username, u.Role)
	}

	// ============================================================
	// 9. 更新关联角色
	// ============================================================
	_, err = queries.UpdateUserNamespaceRole(ctx, generated.UpdateUserNamespaceRoleParams{
		UserID:      user2.ID,
		NamespaceID: ns1.ID,
		Role:        "admin",
	})
	if err != nil {
		log.Fatalf("update role: %v", err)
	}
	fmt.Printf("\nUpdated %s's role in %s to admin\n", user2.Username, ns1.Name)
	// ============================================================
	// 10. 复杂查询：用户列表（筛选 + 排序 + 分页 + 关联 namespace）
	// ============================================================
	activeStatus := "active"
	users, err := queries.ListUsers(ctx, generated.ListUsersParams{
		Status:    &activeStatus,       // 筛选 status = 'active'
		Username:  nil,                 // 不筛选 username
		Email:     nil,                 // 不筛选 email
		DisplayName: nil,              // 不筛选 display_name
		SortField: db.UserSortUsername, // 按 username 排序
		SortOrder: db.SortAsc,         // 升序
		PageSize:  10,                 // 每页 10 条
		PageOffset: 0,                 // 第一页
	})
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	fmt.Printf("\nListUsers (active, sort by username asc, page 1):\n")
	for _, u := range users {
		fmt.Printf("  - %s <%s> namespaces=%v\n", u.Username, u.Email, u.NamespaceNames)
	}

	// 带模糊搜索的查询（使用 EscapeLike 防止 LIKE 注入）
	searchName := db.EscapeLike("ali")
	users, err = queries.ListUsers(ctx, generated.ListUsersParams{
		Username:    &searchName,
		SortField:   db.UserSortCreatedAt,
		SortOrder:   db.SortDesc,
		PageSize:    10,
		PageOffset:  0,
	})
	if err != nil {
		log.Fatalf("list users with search: %v", err)
	}
	fmt.Printf("\nListUsers (username like '%%ali%%'):\n")
	for _, u := range users {
		fmt.Printf("  - %s <%s>\n", u.Username, u.Email)
	}

	// 获取总数（用于分页）
	count, err := queries.CountUsers(ctx, generated.CountUsersParams{
		Status: &activeStatus,
	})
	if err != nil {
		log.Fatalf("count users: %v", err)
	}
	fmt.Printf("\nTotal active users: %d\n", count)

	// ============================================================
	// 11. 复杂查询：Namespace 列表（筛选 + 排序 + 分页 + 关联 owner）
	// ============================================================
	privateVis := "private"
	namespaces, err := queries.ListNamespaces(ctx, generated.ListNamespacesParams{
		Status:     &activeStatus,
		Visibility: &privateVis,        // 筛选 visibility = 'private'
		SortField:  db.NamespaceSortName,
		SortOrder:  db.SortAsc,
		PageSize:   10,
		PageOffset: 0,
	})
	if err != nil {
		log.Fatalf("list namespaces: %v", err)
	}
	fmt.Printf("\nListNamespaces (active + private, sort by name):\n")
	for _, ns := range namespaces {
		fmt.Printf("  - %s (owner: %s, visibility: %s)\n", ns.Name, ns.OwnerUsername, ns.Visibility)
	}
	// ============================================================
	// 12. 事务示例：在事务中创建 Namespace 并添加成员
	// ============================================================
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	// 用事务创建 Queries
	qtx := queries.WithTx(tx)

	ns3, err := qtx.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        "team-gamma",
		DisplayName: "Team Gamma",
		Description: "Created in a transaction",
		OwnerID:     user1.ID,
		Visibility:  "public",
		MaxMembers:  200,
		Status:      "active",
	})
	if err != nil {
		tx.Rollback(ctx)
		log.Fatalf("create namespace in tx: %v", err)
	}

	_, err = qtx.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      user1.ID,
		NamespaceID: ns3.ID,
		Role:        "owner",
	})
	if err != nil {
		tx.Rollback(ctx)
		log.Fatalf("add user to namespace in tx: %v", err)
	}

	_, err = qtx.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      user2.ID,
		NamespaceID: ns3.ID,
		Role:        "member",
	})
	if err != nil {
		tx.Rollback(ctx)
		log.Fatalf("add user to namespace in tx: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit tx: %v", err)
	}
	fmt.Printf("\nTransaction committed: created namespace %s with 2 members\n", ns3.Name)

	// ============================================================
	// 13. 删除关联 & 删除实体
	// ============================================================
	err = queries.RemoveUserFromNamespace(ctx, generated.RemoveUserFromNamespaceParams{
		UserID:      user2.ID,
		NamespaceID: ns1.ID,
	})
	if err != nil {
		log.Fatalf("remove user from namespace: %v", err)
	}
	fmt.Printf("\nRemoved %s from %s\n", user2.Username, ns1.Name)

	memberCount, err := queries.CountUsersByNamespaceID(ctx, ns1.ID)
	if err != nil {
		log.Fatalf("count members: %v", err)
	}
	fmt.Printf("Namespace %s now has %d member(s)\n", ns1.Name, memberCount)

	fmt.Println("\nAll examples completed successfully!")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
