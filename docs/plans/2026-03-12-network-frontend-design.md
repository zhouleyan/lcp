# Network Module Frontend Design

## Context

Backend API for Network module (Phase 1) is complete on `feat/network` branch. This design covers the frontend implementation following existing IAM/Infra module patterns.

## Page Structure

| Route | Page | Features |
|-------|------|----------|
| `/network/networks` | Network List | Search/filter/sort/pagination + create/edit/batch delete |
| `/network/networks/:networkId` | Network Detail | Info card + Subnets Tab (subnet table + create/delete) |
| `/network/networks/:networkId/subnets/:subnetId` | Subnet Detail | Info card + IP stats bar + Allocations Tab (IP table + allocate/release) |

## Component Design

- **Network List**: Standard list page (name, displayName, status, subnetCount, createdAt). Reuses `useListState` + pagination/sort/search.
- **Network Detail**: Info card with edit + Tab container. Subnets Tab embeds subnet table (cidr, gateway, freeIPs/usedIPs, status).
- **Subnet Detail**: Info card (with IP usage progress bar: used/total) + Allocations Tab (ip, description, isGateway badge, createdAt).
- **IP Allocation Dialog**: Input IP + description, frontend IP format validation.
- **IP Release**: Confirm dialog, gateway rows have delete button disabled.

## API Layer

```
src/api/network/
  client.ts      — networkApi = api.extend({ prefixUrl: '/api/network/v1' })
  networks.ts    — CRUD + batch delete
  subnets.ts     — CRUD + batch delete (scoped to networkId)
  allocations.ts — list + create + delete (scoped to networkId + subnetId)
```

## i18n

New locale files: `src/i18n/locales/{en-US,zh-CN}/network.ts` with network/subnet/allocation translation keys.

## Routing & Navigation

- New `src/pages/network/routes.tsx`
- `nav-config.ts`: Add Network nav group (icon: `Network` from lucide-react)
- `modules.ts`: Add `"network"` prefix
- `routes.tsx`: Register network routes

## Patterns Followed

- Same list/detail/form patterns as IAM workspaces (detail page with tabs)
- Same API client pattern (ky extend with prefixUrl)
- Same i18n key structure as infra module
- Same permission hook usage for button-level access control
