# Approval Model Explained — What Users Can and Cannot Do

This document explains how user actions, approvals, and custody controls work in this platform.

The key idea:
**Users can request actions, but custody controls execution.**

---

## Do End Users Need Access to the Custodian UI?

Typically, **no**.

End users interact only with this platform. They do not need direct access to the custodian’s interface.

However, **human approvals are still required**, and those are performed by designated approvers (operators, compliance officers, or customer admins).

---

## Warm Wallet Approvals

### What Users Can Do

- Request transfers
- See when approval is required
- Track approval status in real time

### What Users Cannot Do

- Approve a transaction simply by clicking a button
- Bypass approval policies
- Trigger automatic signing

### How Approval Actually Happens

- A designated approver reviews the request
- Approval is performed using custody-controlled identity and security
- Once approved, the transaction proceeds automatically

Your app shows the result, not the signing step.

---

## Cold Wallet Transfers

### What Users Can Do

- Request transfers from cold storage
- Track the custody workflow status
- Receive notifications when completed

### What Users Cannot Do

- Trigger cold signing themselves
- Force execution timing
- Expect real-time completion

Cold storage involves offline processes by design.

---

## Why This Model Exists

Custodial security relies on:

- Separation of duties
- Human-in-the-loop approvals
- Offline controls for cold storage

Allowing end users to auto-approve or auto-sign would defeat custody guarantees.

---

## Best-Practice UX Pattern

- End users: request and observe
- Approvers: approve via custody controls
- Platform: orchestrates, tracks, and informs

This ensures:

- Strong security
- Clear accountability
- Regulatory compliance

---

## Summary

- End users do **not** need custody UI access
- End users **cannot** bypass human approvals
- Your platform provides a unified experience
- Custody controls remain intact

Security is enforced by design, not by trust.
