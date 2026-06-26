# Errors And Retries

Status: Step 12 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Production Kuafu needs consistent error semantics before it talks to Kubernetes, MongoDB, and scheduler adapters.

## Current Implementation

- [internal/errors](../internal/errors) defines typed error kinds: transient, permanent, not found, quota.
- [internal/retry](../internal/retry) defines a small context-aware retry helper.
- [internal/controller/sync_controller.go](../internal/controller/sync_controller.go) retries transient runtime adapter submit/status failures.
- Permanent errors are not retried.

## Rules

- Transient errors are retryable: temporary API outage, network timeout, leader change.
- Permanent errors are not retryable: invalid job spec, unsupported scheduler feature.
- Quota errors are not retried by the controller; users or policy must change input.
- Not found errors are interpreted by the caller based on ownership context.

## Exit Criteria

Step 12 scaffold is complete when:

- Typed error kinds are available.
- Retry helper is context-aware and tested.
- Sync controller retries transient adapter failures and does not retry permanent failures.
- Production startup remains fail-fast until real runtime wiring exists.
