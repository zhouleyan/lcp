# End-to-End API Design: User & Namespace

**Date:** 2026-02-27
**Status:** Approved

## Overview

Implement end-to-end REST API flow: HTTP request → validation → service → store → database.
Uses K8s-style API objects (TypeMeta + ObjectMeta + Spec) with hand-written validation functions.

## Architecture

```
HTTP Request
    ↓
Handler (route + request parsing + response serialization)
    ↓
Validation (hand-written functions, return field errors)
    ↓
Service (orchestrate business logic, call store, manage transactions)
    ↓
Store (data access, existing implementation)
    ↓
Database (PostgreSQL via sqlc)
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /apis/v1/users | Create user |
| GET | /apis/v1/users/{userId} | Get user by ID |
| POST | /apis/v1/namespaces | Create namespace |
| GET | /apis/v1/namespaces/{namespaceId} | Get namespace by ID |
| POST | /apis/v1/namespaces/{namespaceId}/members | Add member to namespace |

## API Types (lib/api/types/)

### ObjectMeta

```go
type ObjectMeta struct {
    ID        string     `json:"id,omitempty"`
    Name      string     `json:"name,omitempty"`
    CreatedAt *time.Time `json:"createdAt,omitempty"`
    UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}
```

### User

```go
type User struct {
    runtime.TypeMeta `json:",inline"`
    ObjectMeta       `json:"metadata"`
    Spec             UserSpec `json:"spec"`
}

type UserSpec struct {
    Username    string `json:"username"`
    Email       string `json:"email"`
    DisplayName string `json:"displayName,omitempty"`
    Phone       string `json:"phone,omitempty"`
    AvatarURL   string `json:"avatarUrl,omitempty"`
    Status      string `json:"status,omitempty"`
}

type UserList struct {
    runtime.TypeMeta `json:",inline"`
    Items            []User `json:"items"`
    Total            int64  `json:"total"`
}
```

### Namespace

```go
type Namespace struct {
    runtime.TypeMeta `json:",inline"`
    ObjectMeta       `json:"metadata"`
    Spec             NamespaceSpec `json:"spec"`
}

type NamespaceSpec struct {
    DisplayName string `json:"displayName,omitempty"`
    Description string `json:"description,omitempty"`
    OwnerID     string `json:"ownerId"`
    Visibility  string `json:"visibility,omitempty"`
    MaxMembers  int    `json:"maxMembers,omitempty"`
    Status      string `json:"status,omitempty"`
}

type NamespaceMember struct {
    runtime.TypeMeta `json:",inline"`
    Spec             NamespaceMemberSpec `json:"spec"`
}

type NamespaceMemberSpec struct {
    UserID string `json:"userId"`
    Role   string `json:"role"`
}
```

## Validation (lib/api/validation/)

Hand-written validation functions returning structured FieldError lists.

### User validation rules

- **username**: required, 3-50 chars, alphanumeric + underscore only
- **email**: required, valid email (net/mail.ParseAddress)
- **phone**: optional, E.164 format (+country code + digits, 7-15 digits)
- **displayName**: optional, max 100 chars
- **status**: optional, must be "active" or "inactive"

### Namespace validation rules

- **name** (ObjectMeta): required, 3-50 chars, lowercase alphanumeric + hyphen
- **ownerId**: required, valid UUID
- **visibility**: optional, "public" or "private" (default "private")
- **maxMembers**: optional, > 0

### NamespaceMember validation rules

- **userId**: required, valid UUID
- **role**: required, one of "admin", "member", "viewer"

## Error Handling (lib/api/errors/)

```go
type StatusError struct {
    Status  int    `json:"status"`
    Reason  string `json:"reason"`
    Message string `json:"message"`
    Details any    `json:"details,omitempty"`
}
```

| Error Type | HTTP Status | When |
|-----------|-------------|------|
| BadRequest | 400 | Validation failure |
| NotFound | 404 | Resource not found |
| Conflict | 409 | Unique constraint violation |
| InternalError | 500 | Unexpected error |

## Service Layer (lib/service/)

Service holds `store.Store`, orchestrates: validation → precondition checks → store calls.

### UserService

- **CreateUser**: validate spec → store.Users().Create() → convert to API type
- **GetUser**: store.Users().GetByID() → convert to API type

### NamespaceService

- **CreateNamespace**: validate → verify owner exists → store.Namespaces().Create()
- **GetNamespace**: store.Namespaces().GetByID() → convert
- **AddMember**: validate → verify user exists → verify namespace exists → store.UserNamespaces().Add()

## REST Helpers (lib/rest/handler.go)

Extend existing handlers with:

- **CreateResource**: parse body → call creator func → return 201
- **readBody**: deserialize request body using content negotiation

## Handler Layer (app/lcp-server/handler/)

- **userHandler**: wraps service.UserService, provides Create/Get functions
- **namespaceHandler**: wraps service.NamespaceService, provides Create/Get/AddMember

## Project Structure

```
lib/
├── api/
│   ├── types/           # API object types
│   │   ├── meta.go      # ObjectMeta
│   │   ├── user.go      # User, UserSpec, UserList
│   │   └── namespace.go # Namespace, NamespaceMember
│   ├── validation/      # Validation functions
│   │   ├── errors.go    # FieldError, ErrorList
│   │   ├── user.go      # ValidateUserCreate
│   │   └── namespace.go # ValidateNamespaceCreate, ValidateNamespaceMember
│   └── errors/          # HTTP error types
│       └── errors.go    # StatusError, NewBadRequest, NewNotFound, etc.
├── service/             # Business service layer
│   ├── service.go       # Service entry
│   ├── user.go          # UserService
│   └── namespace.go     # NamespaceService
├── rest/                # Existing - extend
│   └── handler.go       # Add CreateResource, readBody
├── store/               # Existing - no changes
└── db/                  # Existing - no changes

app/lcp-server/handler/
├── handler.go           # Route registration - extend
├── user.go              # userHandler (new)
└── namespace.go         # namespaceHandler (new)
```
