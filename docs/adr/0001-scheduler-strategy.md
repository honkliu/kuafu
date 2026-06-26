# ADR 0001: Scheduler Adapter Strategy

Status: proposed  
Date: 2026-06-25

## Context

Kuafu needs production GPU scheduling without building a scheduler from scratch. OpenPAI provides FrameworkController and HiveD patterns; Volcano is an active Kubernetes-native fallback.

A00/A01 Kubernetes bootstrap is currently blocked by running Docker containers and `live-restore=false`, so live scheduler deployment is deferred until a maintenance window.

## Decision

Use a scheduler adapter boundary:

- Keep `RuntimeAdapter` as the API/controller-facing contract.
- Treat FrameworkController + HiveD as the first topology-aware spike once Kubernetes is available.
- Keep Volcano as the first fallback spike.
- Build adapter translation logic with pure unit tests before wiring live Kubernetes clients.

## Consequences

- Kuafu can continue implementation without pretending production scheduling is live.
- FrameworkController/HiveD/Volcano adapters can be evaluated independently.
- Runtime object translation can be tested before A00/A01 mutation is allowed.

## Current Evidence

- `RuntimeJobSpec` is scheduler-neutral.
- `UnsupportedRuntimeAdapter` keeps production fail-fast honest.
- Volcano manifest translation exists as a non-mutating scaffold.
