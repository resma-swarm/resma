# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- RBAC with 3 roles (owner/admin/user) and RequireRole middleware
- User management UI (CRUD) with onboarding flow creating 'owner'
- API Keys management UI with One-Time Read (OTR) plaintext reveal
- Settings area with nested routes (users, api-keys, parameters, data)
- Two-tier config (env var infra + DB operacional via /api/settings)
- Data retention expansion (task_history, volume_metrics, storage_summary, change_log)
- Stale-marking for services and nodes (soft delete, status='stale')
- Data prune endpoints with dry-run and audit logging
- shadcn official Sidebar migration (NavUser, NavSettings, NavMain)
- Profile page with change password
- StaleServiceDays config (RESMA_STALE_SERVICE_DAYS, default 7)
- CleanupExpiredRefreshTokens in retention loop
### Changed
- Onboarding now creates 'owner' instead of 'admin'
- Layout migrated from custom to shadcn Sidebar (SidebarProvider + SidebarInset)
- RunRetention expanded from 3 to 7 tables
- dropdown-menu and tabs replaced with official shadcn versions
- avatar and select replaced with official shadcn versions
### Security
- RequireRole middleware protects all write endpoints (schedules, templates, services, recommendations apply, api-keys, users, settings, prune)
- API keys plaintext shown only once at creation (OTR)
- Owner role is unique (cannot be created via UI, only via onboarding)
- Owner cannot be deleted or have role changed
- Users cannot delete themselves
