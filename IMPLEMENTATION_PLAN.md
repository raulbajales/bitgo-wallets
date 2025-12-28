# Implementation Plan — Custodial Wallet Platform (Warm & Cold)

This document describes the implementation plan for a platform that allows users to manage Warm and Cold custodial wallets backed by BitGo Custody.

The system consists of:

- A **Next.js web app** (user-facing)
- A **Go REST API** (backend orchestration + BitGo integration)

The platform supports **request + tracking workflows**, while respecting custody constraints.

---

## Milestone 0 — Architecture & Scope Definition

### Goals

- Align product expectations with BitGo Custody constraints
- Define supported Warm (A) and Cold (B) use cases
- Establish clear trust boundaries

### Tasks

- Define supported flows:
  - Warm: request → submit → pending approval → completion
  - Cold: request → offline workflow → completion
- Define roles:
  - End User
  - Operator
  - Approver
  - Admin
- Define canonical transfer status model
- Produce API contracts between Web ↔ Go API
- Write internal “capabilities vs custody limits” doc

---

## Milestone 1 — Foundations & Security Baseline

### Goals

- Secure, observable, production-ready base

### Tasks

- Create repositories:
  - `web/` (Next.js)
  - `api/` (Go)
- Secure secret handling (BitGo tokens only in Go API)
- Add health checks and structured logging
- Correlation IDs across Web → API → BitGo

---

## Milestone 2 — Data Model & Persistence

### Goals

- Track users, wallets, and custody workflows

### Tasks

- Choose database (PostgreSQL)
- Define schema:
  - users
  - tenants / organizations
  - wallets (BitGo walletId mapping)
  - wallet_memberships
  - transfer_requests
  - audit_logs
- Implement migrations
- Define canonical status lifecycle
- CRUD APIs for wallets and transfers

---

## Milestone 3 — BitGo Integration Layer (Go)

### Goals

- Safe, reusable BitGo API abstraction

### Tasks

- Implement BitGo client:
  - Auth headers
  - Retries & timeouts
  - Redacted logging
- Wallet discovery APIs
- Transfer build / submit adapters
- Status normalization
- Idempotency handling

---

## Milestone 4 — Warm Wallet Workflows (A)

### Goals

- End-to-end Warm wallet transfer requests

### Tasks (Backend)

- Endpoints:
  - Build transfer
  - Submit transfer
  - Get transfer status
  - List transfers
- Map BitGo pending approvals → UI states
- Polling worker
- Notifications for pending approvals

### Tasks (Frontend)

- login form using expecting dummy email and pass
- Wallet dashboard
- Create transfer form
- Transfer detail page with status timeline
- “Pending approval” UX with no auto-approval

---

## Milestone 5 — Cold Wallet Workflows (B)

### Goals

- Cold storage request + lifecycle tracking

### Tasks (Backend)

- Cold transfer request endpoint
- Enforce stricter validation and allowlists
- Track offline custody workflow states
- SLA-aware status handling

### Tasks (Frontend)

- Cold transfer request UI
- Explicit “not instant” messaging
- Cold transfer timeline view
- Admin queue for cold requests

---

## Milestone 6 — Governance & Audit

### Goals

- Enterprise-grade controls

### Tasks

- Role-based access control
- Velocity and risk limits
- Comprehensive audit logging
- Transfer reconciliation jobs
- Exportable reports

---

## Milestone 7 — Production Hardening

### Goals

- Reliability, observability, and security

### Tasks

- Load testing
- Rate limiting
- Alerts for stuck workflows
- Webhook replay support
- Backups and recovery plan

---

## Milestone 8 — Platform Readiness

### Goals

- Multi-tenant, operator-ready product

### Tasks

- Tenant isolation
- Admin tooling
- Wallet provisioning workflows
- Operational dashboards

---
