# Account Usage Visibility Design

## Goal

Make the user-facing `使用记录` page show the upstream account name for each call record, and add an account-dimension consumption table to the admin dashboard so operators can view account-level token usage, account cost, and request count within a selected time range.

## Scope

This design covers two UI surfaces:

1. User-facing usage records at [frontend/src/views/user/UsageView.vue](/Users/chenxiaoming/work/sub2api/frontend/src/views/user/UsageView.vue)
2. Admin dashboard at [frontend/src/views/admin/DashboardView.vue](/Users/chenxiaoming/work/sub2api/frontend/src/views/admin/DashboardView.vue)

This design intentionally adopts Option A:

- Regular users will see the real upstream account name in their own usage records.
- The admin dashboard will show account-level aggregated consumption using account billing semantics, not user billing semantics.

## Current State

### User Usage Records

- The user-facing usage DTO does not include any account summary.
- The user usage page renders columns for API key, model, endpoint, request type, tokens, cost, and time.
- CSV export from the same page follows the same omission and does not include account name.

### Admin Usage and Dashboard

- The admin usage log table already shows account name, so there is no missing backend path there.
- The admin dashboard already supports date-range-driven charts and ranking data.
- The admin dashboard API already has batch aggregation patterns for users and API keys, but no equivalent account-dimension endpoint.

## Product Decisions

### User-Facing Account Exposure

The user usage page will display the actual account name that handled the request.

Rationale:

- This is explicitly requested behavior.
- The data already exists in the usage log domain model and admin DTO path.
- The change is presentation-focused and does not alter request routing or billing.

Trade-off:

- This exposes internal upstream account names to regular users.
- That exposure is accepted for this feature slice and is not obfuscated in this design.

### Account Cost Semantics

The admin dashboard account table will use account billing semantics for the amount column.

Definition:

- Per log row, account amount = `(account_stats_cost ?? total_cost) * (account_rate_multiplier ?? 1)`
- Per account aggregate, account amount = sum of the above value over the selected time range

Rationale:

- This matches the existing "账号计费" semantics already shown in admin usage records and account-related views.
- It avoids mixing operator-facing account cost with user-facing actual billed cost.

## Design

### 1. User Usage Records: Add Account Name Column

#### API / DTO changes

- Extend the user usage DTO to optionally include a minimal account summary with `id` and `name`.
- Reuse the existing account summary shape used by admin usage when practical, rather than creating a second parallel shape with different field names.
- Ensure the user usage list endpoint returns the hydrated account summary when an account is attached to the usage log.

#### Frontend changes

- Add an `account` column to the user usage table in [frontend/src/views/user/UsageView.vue](/Users/chenxiaoming/work/sub2api/frontend/src/views/user/UsageView.vue).
- Render `row.account?.name || '-'`.
- Keep the new column display-only:
  - no new filter
  - no click action
  - no sort requirement for this slice

#### Export behavior

- Add account name to the CSV export generated from the same page.
- Keep export column order aligned with the table order as much as practical to reduce surprise.

### 2. Admin Dashboard: Add Account Consumption Table

#### UX

Add a new card below the existing chart area in [frontend/src/views/admin/DashboardView.vue](/Users/chenxiaoming/work/sub2api/frontend/src/views/admin/DashboardView.vue).

The card contains a plain table with four columns:

1. `账号名称`
2. `调用次数`
3. `总 Tokens`
4. `账号金额`

Behavior:

- Driven by the same date range already selected on the dashboard
- Loaded independently from existing charts and rankings
- Default sort: `账号金额` descending
- Empty state: reuse existing neutral empty-state styling or simple no-data messaging

This table does not need:

- pagination for the first slice
- inline filtering
- row click behavior
- chart visualization

#### Data returned

Each row should include:

- `account_id`
- `account_name`
- `request_count`
- `total_tokens`
- `account_cost`

### 3. Admin Dashboard API

Add a dedicated account-dimension dashboard endpoint under the existing admin dashboard route family.

Recommended route:

- `GET /api/v1/admin/dashboard/accounts`

Query parameters:

- `start_date`
- `end_date`

Response shape:

- `accounts`: account aggregate rows
- `start_date`
- `end_date`

Recommended response row shape:

- `account_id`
- `account_name`
- `request_count`
- `total_tokens`
- `account_cost`

Rationale for a dedicated endpoint instead of overloading snapshot or ranking APIs:

- The table has a distinct data shape.
- It keeps the existing dashboard snapshot lean.
- It mirrors the current dashboard API style where specialized widgets have dedicated endpoints.

### 4. Repository / Service Aggregation

Implement a repository aggregation method that groups usage logs by `account_id` over a selected range.

Aggregate fields:

- request count: `COUNT(*)`
- total tokens: sum of `input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens`
- account cost: sum of `(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1))`

Additional behavior:

- Join or hydrate account name so the frontend does not need a second lookup.
- Exclude rows with missing or zero account IDs.
- Keep results sorted by account cost descending in SQL if convenient; otherwise sort in service/handler before response.

The service layer should be a thin pass-through, matching the current dashboard service style.

## Data Flow

### User Usage Flow

1. Usage list query loads usage logs for the current user.
2. Backend mapper includes minimal account summary in each DTO.
3. Frontend renders the new account column.
4. CSV export uses the same DTO field.

### Admin Dashboard Flow

1. Admin dashboard date range changes.
2. Frontend issues a request to the new account aggregation endpoint.
3. Backend repository aggregates usage logs by account within the date range.
4. Handler returns normalized rows with account name and metrics.
5. Frontend renders the table sorted by account cost descending.

## Error Handling

### User Usage Page

- If the account summary is absent for a log row, show `-`.
- The page should keep working if some historical logs reference deleted or unhydrated accounts.

### Admin Dashboard

- If the account aggregation request fails, keep the rest of the dashboard functional.
- Show an inline failure or empty state in the account table card rather than blocking the entire dashboard page.
- If an account has been deleted but logs remain, return a fallback name such as `#<account_id>` or `-` depending on available data.

## Performance Considerations

- The account dashboard table is bounded by time range and account cardinality, which should be materially smaller than raw log listing.
- A dedicated aggregate query is preferable to fetching raw logs and aggregating in memory.
- No caching is required for the first slice, but the endpoint should remain compatible with adding the same lightweight snapshot cache pattern later if needed.

## Testing Strategy

### Backend

1. DTO mapper test:
   - user usage DTO includes account summary when present
2. User/API contract coverage:
   - user usage list response serializes account data correctly
3. Repository aggregation test:
   - account grouping returns expected request count, total tokens, and account cost
   - account cost uses `account_stats_cost` when present and falls back to `total_cost` otherwise
4. Admin handler test:
   - validates date range parsing and response shape for the new endpoint

### Frontend

1. User usage page test:
   - renders account name column when account exists
   - renders `-` when account is absent
2. User export test:
   - exported rows include account name
3. Admin dashboard test:
   - renders the new account table rows from API data
   - handles empty state
   - preserves existing dashboard sections when the new request fails

## Out of Scope

- Adding account-based filtering to the user usage page
- Obfuscating or aliasing account names
- Adding pagination, searching, or sorting controls to the new admin dashboard table
- Folding account aggregation into snapshot-v2
- Changing existing billing semantics outside this new table

## Implementation Notes

- Prefer extending existing usage DTO/account summary types instead of inventing near-duplicate types.
- Keep the admin dashboard table isolated in its own component if the parent view starts to get crowded; otherwise a small inline table is acceptable if it remains readable.
- Keep the new backend endpoint consistent with current dashboard handler/service/repository layering.
