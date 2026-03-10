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
npm run dev          # Start dev server (port 5173)
npm run build        # Production build
npx tsc --noEmit     # Type check
```

## Directory Structure

```
src/
  api/           # API client, type definitions, per-resource API functions
  components/    # Shared components (user-menu, ui/ for shadcn primitives)
  i18n/          # i18n setup, locale files (en-US, zh-CN), type definitions
  lib/           # Auth utilities (OIDC PKCE flow, token management)
  pages/         # Page components by route
  stores/        # Zustand stores (auth-store)
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
- Locale files: `src/i18n/locales/en-US.ts` and `zh-CN.ts`
- Key naming: `{domain}.{subkey}` (e.g., `user.create`, `api.validation.required`, `common.save`)
- Parameterized messages use `{param}` syntax: `t("api.validation.required", { field: t("common.phone") })`
- Both locale files must always have the same keys (typed via `Messages`)

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

## API Client Conventions

- Use `ky` via the shared `api` instance (`api/client.ts`) which handles auth tokens and 401 refresh
- Wrap all API calls with `apiRequest()` to convert HTTP errors to `ApiError`
- `ApiError` carries `status`, `reason`, `message`, and optional `details[]`
