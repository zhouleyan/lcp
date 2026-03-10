# UI CLAUDE.md

## Tech Stack

- React 18 + TypeScript + Vite
- Tailwind CSS + shadcn/ui (Radix primitives)
- react-hook-form + zod/v4 + @hookform/resolvers/zod
- ky (HTTP client)
- zustand (state management)
- react-i18next pattern via custom `useTranslation` hook

## Key Commands

```bash
pnpm dev             # Start dev server (port 5173)
pnpm build           # Production build
pnpm test            # Run unit tests (vitest)
npx tsc --noEmit     # Type check
```

## Directory Structure

```
src/
  api/              # API client, shared types
  api/iam/          # IAM module API functions (users, workspaces, namespaces, rbac)
  components/       # Shared components (user-menu, scope-selector, permission-selector, ui/ for shadcn primitives)
  hooks/            # Custom hooks (use-permission, use-list-state)
  i18n/             # i18n setup, type definitions
  i18n/locales/     # Per-module locale files (en-US/, zh-CN/ each with common, iam, dashboard, audit)
  lib/              # Auth utilities (OIDC PKCE flow, token management)
  modules.ts        # Module definitions (iam, dashboard) with nav items and routes
  pages/iam/        # IAM module pages (users, workspaces, namespaces, roles)
  pages/dashboard/  # Dashboard module pages (overview)
  routes.tsx        # Route definitions with module-based lazy loading
  stores/           # Zustand stores (auth-store, permission-store, scope-store, workspace-store)
  test/             # Test setup (vitest)
```

## Form Validation Conventions

### Architecture: Frontend & Backend Dual Validation

Frontend validation provides instant UX feedback; backend validation is the authoritative source. Both layers must enforce the same rules. Validation rules are defined independently on each side — frontend in zod schemas, backend in Go `validation.go`.

### Frontend Validation (zod + react-hook-form)

#### useForm Configuration

```tsx
const form = useForm<FormValues>({
  resolver: zodResolver(schema),
  mode: "onBlur",              // First validation triggers on blur (not on focus/click)
  // Do NOT set reValidateMode: "onBlur" — default "onChange" ensures errors
  // clear immediately as the user types corrections after an initial error.
  defaultValues: { ... },
})
```

- `mode: "onBlur"` — validates when the field loses focus, not while typing
- `reValidateMode` — keep default (`"onChange"`). Setting `"onBlur"` causes stale errors to persist until the next blur, confusing users who fix input but still see the old error

#### Dialog Forms

Prevent Radix Dialog auto-focus from triggering validation on open:

```tsx
<DialogContent onOpenAutoFocus={(e) => e.preventDefault()}>
```

Reset form state when dialog closes:

```tsx
<Dialog onOpenChange={(v) => { onOpenChange(v); if (!v) form.reset() }}>
```

#### Zod Schema Pattern

Define schemas inside the component (not at module level) to access `t()` for i18n error messages:

```tsx
const schema = z.object({
  username: z.string()
    .min(3, t("api.validation.username.format"))
    .max(50, t("api.validation.username.format"))
    .regex(/^[a-zA-Z0-9_]+$/, t("api.validation.username.format")),
  email: z.email(t("api.validation.email.format")),
  phone: z.string()
    .min(1, t("api.validation.required", { field: t("common.phone") }))
    .regex(/^1[3-9]\d{9}$/, t("api.validation.phone.format")),
  password: z.string()
    .min(8, t("api.validation.password.length"))
    .max(128, t("api.validation.password.length"))
    .regex(/[A-Z]/, t("api.validation.password.uppercase"))
    .regex(/[a-z]/, t("api.validation.password.lowercase"))
    .regex(/[0-9]/, t("api.validation.password.digit")),
})
```

For cross-field validation, use `.refine()` on the object with `path` to target the right field:

```tsx
.refine((data) => data.newPassword !== data.oldPassword, {
  message: t("userMenu.passwordSameAsOld"),
  path: ["newPassword"],
})
```

#### Password Field Hint

Show password rules via `FormDescription` (not as an error):

```tsx
<FormDescription>{t("api.validation.password.hint")}</FormDescription>
```

#### Uniqueness Checks (Frontend-Initiated)

No dedicated uniqueness endpoint exists. Use `listUsers` with a filter, then compare exact match client-side:

```tsx
const checkUniqueness = async (field: "username" | "email" | "phone", value: string) => {
  if (!value) return
  const data = await listUsers({ page: 1, pageSize: 1, [field]: value })
  const exists = data.items?.some((u) => {
    if (isEdit && u.metadata.id === user?.metadata.id) return false
    return u.spec[field]?.toLowerCase() === value.toLowerCase()
  })
  if (exists) form.setError(field, { message: t(`api.validation.${field}.taken`) })
}
```

Trigger uniqueness check on blur, only after format validation passes:

```tsx
onBlur={async (e) => {
  field.onBlur()                            // trigger zod validation
  if (!e.target.value) return
  const valid = await form.trigger("email") // wait for result
  if (valid) checkUniqueness("email", e.target.value)
}}
```

### Backend Validation

Backend returns `{status, reason, message, details}` where `details` is `{field, message}[]`.

#### Validation rules (Go `validation.go`)

| Field    | Rule                                               | Error message                                               |
|----------|-----------------------------------------------------|-------------------------------------------------------------|
| username | required, `^[a-zA-Z0-9_]{3,50}$`                   | `is required` / `must be 3-50 alphanumeric characters or underscores` |
| email    | required, RFC 5322 (`mail.ParseAddress`)            | `is required` / `is not a valid email address`              |
| phone    | required, `^1[3-9]\d{9}$`                           | `is required` / `must be a valid Chinese mobile number (e.g. 13800138000)` |
| password | 8-128 chars, `[A-Z]`, `[a-z]`, `[0-9]`             | `must be 8-128 characters` / `must contain at least one ...` |
| status   | `active` or `inactive`                              | `must be 'active' or 'inactive'`                            |

Uniqueness is enforced by PostgreSQL UNIQUE constraints (username, email, phone). Conflict → 409 with `reason: "Conflict"`.

### Error Translation Chain (api/client.ts)

Backend returns English error messages. The frontend translates them to i18n keys for display:

1. **Field-level errors** (`details[].message`): matched via `detailMessageMap` → i18n key → `t(key, { field })`
2. **API-level errors** (`message`): matched via `messageMap` (exact message match)
3. **Reason-level errors** (`reason`): matched via `reasonMessageMap` (e.g., `Conflict` → `api.error.conflict`)
4. **Fallback**: show raw backend message

#### Adding a new backend error message

1. Backend: add the English message string in `validation.go` or storage layer
2. Frontend `api/client.ts`: add mapping in `detailMessageMap`, `messageMap`, or `reasonMessageMap`
3. i18n locale files: add the i18n key to both `en-US.ts` and `zh-CN.ts`

### Error Display Pattern

- **Field-level errors**: use `form.setError(field, { message })` → rendered by `<FormMessage />`
- **Form-level errors**: use `form.setError("root", { message })` → rendered by a banner at the top of the form:

```tsx
{form.formState.errors.root && (
  <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
    {form.formState.errors.root.message}
  </div>
)}
```

- **Non-form actions** (delete, batch operations): use `toast.error()` / `toast.success()`

## I18n Conventions

- All user-visible text must use `t("key")` — no hardcoded strings
- Locale files are split by module under `src/i18n/locales/{locale}/`:
  - `common.ts` — common.*, auth.*, login.*, nav.*, scope.*, overview.*, error.*, api.*, action.*, userMenu.*, perm.group.all, perm.verb.*, perm.verbGroup.*
  - `iam.ts` — workspace.*, namespace.*, user.*, role.*, rolebinding.*, perm.group.iam.*, perm.iam:*
  - `dashboard.ts` — perm.group.dashboard.*, perm.dashboard:*
  - `audit.ts` — audit.*, perm.group.audit.*, perm.audit:*
  - `index.ts` — re-exports merged messages
- When adding a new module, create a new `{module}.ts` file in each locale directory and import it in `index.ts`
- Key naming: `{domain}.{subkey}` (e.g., `user.create`, `api.validation.required`, `common.save`)
- Parameterized messages use `{param}` syntax: `t("api.validation.required", { field: t("common.phone") })`
- Both locale directories must always have the same keys (typed via `Messages`)

## List/Table Conventions

All resource list views — whether top-level pages (Users, Workspaces, Namespaces) or embedded sub-lists (workspace members, namespace members) — must provide a consistent, full-featured table experience:

- **Search**: Debounced text input (300ms), filters across name/email/phone/displayName
- **Status filter**: Dropdown on the Status column header (All / Active / Inactive)
- **Sortable columns**: Click column header to toggle asc/desc, with sort icons (ArrowUpDown / ArrowUp / ArrowDown)
- **Pagination**: Page size selector (10/20/50/100), prev/next buttons, "Page X of Y" display
- **Multi-select**: Checkbox column for batch operations (batch delete / batch remove)
- **Loading skeleton**: Show skeleton rows while fetching
- **Empty state**: Centered message when no data

This applies equally to sub-resource lists embedded in detail pages (e.g., members tab in workspace detail). Do not create simplified list views — every list table must have the full feature set above.

### useCallback dependency: never include `t`

The `useTranslation()` hook returns a new `t` function reference on every render. Including `t` in a `useCallback` dependency array causes infinite re-render loops (new `t` → new callback → useEffect fires → setState → re-render → new `t` → ...). Always exclude `t` from `useCallback` deps:

```tsx
const fetchData = useCallback(async () => {
  // ... can use t() inside for error messages
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [page, pageSize, sortBy, sortOrder, search, statusFilter]) // NO `t` here
```

## Route-API Alignment Convention

Frontend routes must mirror backend API routes to enable unified RBAC permission control. The same path pattern on both sides allows a single `canAccess(path, action)` function to gate both backend API access and frontend UI visibility.

**Pattern**: If the backend API is `/api/iam/v1/workspaces/{workspaceId}/users`, the frontend route is `/iam/workspaces/:workspaceId/users` (strip the `/api` prefix and version, keep module name).

**Examples**:

| Backend API | Frontend Route |
|---|---|
| `/api/iam/v1/workspaces` | `/iam/workspaces` |
| `/api/iam/v1/workspaces/{workspaceId}` | `/iam/workspaces/:workspaceId` |
| `/api/iam/v1/workspaces/{workspaceId}/users` | `/iam/workspaces/:workspaceId/users` |
| `/api/iam/v1/workspaces/{workspaceId}/namespaces` | `/iam/workspaces/:workspaceId/namespaces` |
| `/api/iam/v1/users` | `/iam/users` |
| `/api/iam/v1/namespaces` | `/iam/namespaces` |

**Key rule**: Use the same resource name on both sides (e.g., `users` not `members`). Frontend routes include the module prefix (`/iam/`) to align with backend API groups. This ensures the permission middleware can match frontend route segments to backend API paths directly.

## Module Architecture

The UI is organized into modules (IAM, Dashboard) defined in `src/modules.ts`. Each module has its own route prefix, navigation items, and lazy-loaded pages.

- **IAM module** (`/iam/`): Users, workspaces, namespaces, roles
- **Dashboard module** (`/dashboard/`): Overview statistics

The root layout (`layouts/root-layout.tsx`) renders navigation items from the active module and integrates the scope selector.

## RBAC & Permissions

### Permission Store (`stores/permission-store.ts`)

Zustand store that fetches the current user's expanded permissions from `/users/{userId}:permissions`. Provides the raw permission data organized by scope (platform, workspaces, namespaces).

### Permission Hook (`hooks/use-permission.ts`)

`usePermission()` hook provides `hasPermission(code)` that checks permissions with scope chain inheritance (platform → workspace → namespace), matching the backend's `HasPermission` logic. The current scope is determined by the `scope-store`.

### Scope Selector (`components/scope-selector.tsx`)

Dropdown that switches between platform/workspace/namespace scope context. Updates `scope-store` which affects permission checks and data filtering across all pages.

**Adding a new resource page**: When adding a page for a new resource type, you MUST register the resource name in `scope-selector.tsx`:
1. Add the resource name to the `KNOWN_RESOURCES` array (e.g., `"rolebindings"`)
2. Add the resource name to the routing conditions in `buildScopedPath` for each applicable scope level (namespace, workspace, platform)

Without this, switching scope while on the new resource page will redirect to the overview page instead of staying on the corresponding resource page at the new scope.

### Permission Selector (`components/permission-selector.tsx`)

Tree-based checkbox component for selecting permission rules when creating/editing roles. Groups permissions by module and resource, supports wildcard patterns, and filters by scope.

### Page-Level Permission Controls

All RBAC pages enforce permission checks at both page and button level using `usePermission()`.

#### Page Guards

Pages that require specific platform-level permissions redirect to `/` if the user lacks access:

```tsx
import { Navigate } from "react-router"
import { usePermission } from "@/hooks/use-permission"
import { usePermissionStore } from "@/stores/permission-store"

const { hasPermission } = usePermission()
const permissionsLoaded = usePermissionStore((s) => s.permissions) !== null

// Only redirect AFTER permissions have loaded (null = still loading)
if (permissionsLoaded && !hasPermission("iam:users:list")) {
  return <Navigate to="/" replace />
}
```

Pages with guards: `users/list` (`iam:users:list`), `roles/list` (`iam:roles:list`).
Pages without guards: workspaces, namespaces (always visible in sidebar).

#### Button-Level Permission Checks

Conditionally render action buttons based on the user's permissions:

```tsx
// Platform-scoped (no scope param): users, roles, rolebindings
{hasPermission("iam:users:create") && <Button>Create</Button>}

// Resource-scoped (per-row): workspaces, namespaces
{hasPermission("iam:workspaces:update", { workspaceId: ws.metadata.id }) && <Button>Edit</Button>}
{hasPermission("iam:namespaces:delete", { namespaceId: ns.metadata.id }) && <Button>Delete</Button>}

// Any-scope (for create button when resources can be created in any scope):
{hasAnyPermission("iam:namespaces:create") && <Button>Create</Button>}
```

#### Permission Code Mapping

| Page | Create | Edit | Delete | Batch Delete |
|------|--------|------|--------|-------------|
| users | `iam:users:create` | `iam:users:update` | `iam:users:delete` | `iam:users:deleteCollection` |
| workspaces | `iam:workspaces:create` | `iam:workspaces:update` (scoped) | `iam:workspaces:delete` (scoped) | `iam:workspaces:deleteCollection` |
| namespaces | `iam:namespaces:create` (any) | `iam:namespaces:update` (scoped) | `iam:namespaces:delete` (scoped) | `iam:namespaces:deleteCollection` (any) |
| workspace members | `iam:workspaces:users:create` (scoped) | - | - | `iam:workspaces:users:deleteCollection` (scoped) |
| namespace members | `iam:namespaces:users:create` (scoped) | - | - | `iam:namespaces:users:deleteCollection` (scoped) |
| roles | `iam:roles:create` | `iam:roles:update` | `iam:roles:delete` | `iam:roles:delete` |
| rolebindings (platform) | `iam:rolebindings:create` | - | `iam:rolebindings:delete` | `iam:rolebindings:delete` |
| rolebindings (workspace) | `iam:workspaces:rolebindings:create` (scoped) | - | `iam:workspaces:rolebindings:delete` (scoped) | same |
| rolebindings (namespace) | `iam:namespaces:rolebindings:create` (scoped) | - | `iam:namespaces:rolebindings:delete` (scoped) | same |

#### Checkbox Column Convention

When batch operations (batch delete, batch remove) are permission-gated, the entire checkbox column (header + per-row cells) must also be wrapped in the same permission check. This avoids showing useless checkboxes to users who can't perform the batch action.

## Testing

- Unit tests via Vitest: `pnpm test`
- Test files: `hooks/__tests__/`, `stores/__tests__/`
- Test setup in `src/test/setup.ts`

## API Client Conventions

- Use `ky` via the shared `api` instance (`api/client.ts`) which handles auth tokens and 401 refresh
- Module-specific API functions live in `api/iam/` (e.g., `rbac.ts`, `users.ts`, `workspaces.ts`, `namespaces.ts`)
- Wrap all API calls with `apiRequest()` to convert HTTP errors to `ApiError`
- `ApiError` carries `status`, `reason`, `message`, and optional `details[]`
