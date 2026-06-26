# API Contract

Status: Step 16 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu needs a stable API contract before frontend, CLI, and production controllers diverge.

## Current Contract

The OpenAPI 3.1 contract is [docs/openapi.yaml](openapi.yaml).

It documents:

- Health and readiness endpoints.
- Node and GPU inventory endpoints.
- Job submit/list/detail/start/stop/restart/cancel/log endpoints.
- Job GPU metrics endpoint for current allocated GPU usage snapshots.
- Node reservation endpoints and lab-mode reserved-node Docker command rendering.
- Queue create/list/detail/update/delete endpoints, including Active/Paused scheduling state.
- Cluster summary endpoint.
- Development-only `X-Kuafu-User` header scaffold for project-scoped job submission.
- A00 external `30000` entry in the server examples.

## Auth Note

The OpenAPI spec documents the current lab/dev header authenticator only so clients know the scaffold behavior. Production mode still fails fast until real OIDC/JWT validation is wired.

## Exit Criteria

Step 16 scaffold is complete when:

- OpenAPI 3.1 YAML exists.
- It covers all current `/api/v1` endpoints.
- It documents current schemas and the dev-only auth surface.
- It includes local and A00 external server examples.
