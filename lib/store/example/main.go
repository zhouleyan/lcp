package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/store"
	pg "lcp.io/lcp/lib/store/pg"
)

func main() {
	ctx := context.Background()

	// 1. 创建连接池
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

	// 2. 创建 Store
	s := pg.New(pool)
	defer s.Close()

	// 3. 创建用户
	alice, err := s.Users().Create(ctx, store.CreateUserParams{
		Username:    "alice",
		Email:       "alice@example.com",
		DisplayName: "Alice Wang",
		Phone:       "13800000001",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create alice: %v", err)
	}
	fmt.Printf("Created user: %s (id=%d)\n", alice.Username, alice.ID)

	bob, err := s.Users().Create(ctx, store.CreateUserParams{
		Username:    "bob",
		Email:       "bob@example.com",
		DisplayName: "Bob Li",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create bob: %v", err)
	}
	fmt.Printf("Created user: %s (id=%d)\n", bob.Username, bob.ID)

	// 4. 事务：创建 Namespace（自动加 owner 为成员）
	var ns *store.Namespace
	err = s.WithTx(ctx, func(txStore store.Store) error {
		var txErr error
		ns, txErr = txStore.Namespaces().Create(ctx, store.CreateNamespaceParams{
			Name:        "team-alpha",
			DisplayName: "Team Alpha",
			Description: "The alpha team",
			OwnerID:     alice.ID,
			Visibility:  "private",
			MaxMembers:  50,
			Status:      "active",
		})
		if txErr != nil {
			return txErr
		}

		// 在同一事务中添加 bob 为成员
		_, txErr = txStore.UserNamespaces().Add(ctx, store.AddUserNamespaceParams{
			UserID:      bob.ID,
			NamespaceID: ns.ID,
			Role:        "member",
		})
		return txErr
	})
	if err != nil {
		log.Fatalf("create namespace in tx: %v", err)
	}
	fmt.Printf("Created namespace: %s (id=%d)\n", ns.Name, ns.ID)

	// 5. 查询用户所属的 Namespace
	nsList, err := s.UserNamespaces().ListByUserID(ctx, alice.ID)
	if err != nil {
		log.Fatalf("list namespaces by user: %v", err)
	}
	fmt.Printf("\n%s belongs to %d namespace(s):\n", alice.Username, len(nsList))
	for _, n := range nsList {
		fmt.Printf("  - %s (role: %s)\n", n.Name, n.Role)
	}

	// 6. 查询 Namespace 下的用户
	members, err := s.UserNamespaces().ListByNamespaceID(ctx, ns.ID)
	if err != nil {
		log.Fatalf("list users by namespace: %v", err)
	}
	fmt.Printf("\n%s has %d member(s):\n", ns.Name, len(members))
	for _, m := range members {
		fmt.Printf("  - %s (role: %s)\n", m.Username, m.Role)
	}

	// 7. 复杂查询：用户列表（筛选 + 排序 + 分页 + 关联 namespace）
	activeStatus := "active"
	result, err := s.Users().List(ctx, store.ListUsersParams{
		Status: &activeStatus,
		Pagination: store.Pagination{
			Page:      1,
			PageSize:  10,
			SortBy:    "username",
			SortOrder: "asc",
		},
	})
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	fmt.Printf("\nListUsers (total=%d):\n", result.TotalCount)
	for _, u := range result.Items {
		fmt.Printf("  - %s <%s> namespaces=%v\n", u.Username, u.Email, u.NamespaceNames)
	}

	// 8. 更新角色
	_, err = s.UserNamespaces().UpdateRole(ctx, store.UpdateRoleParams{
		UserID:      bob.ID,
		NamespaceID: ns.ID,
		Role:        "admin",
	})
	if err != nil {
		log.Fatalf("update role: %v", err)
	}
	fmt.Printf("\nUpdated %s's role to admin\n", bob.Username)

	// 9. 删除成员
	err = s.UserNamespaces().Remove(ctx, bob.ID, ns.ID)
	if err != nil {
		log.Fatalf("remove member: %v", err)
	}
	fmt.Printf("Removed %s from %s\n", bob.Username, ns.Name)

	fmt.Println("\nAll store examples completed successfully!")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
