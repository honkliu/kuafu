# Multi-Tenancy

Status: Step 14 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu needs project-scoped resource ownership before production auth/RBAC is wired.

## Current Model

- `Project` identifies a tenant/project and its default queue.
- `User` identifies a person/service account by alias and project memberships.
- `ProjectRole` supports `Admin`, `Submitter`, and `Viewer`.
- `Queue.Project` scopes a queue to a project.
- `Job.Project` records the submitting project.

## Current Policy

- Project-scoped queues reject jobs with missing or mismatched project.
- `CanSubmit` allows project admins and submitters.
- Viewers cannot submit.

## Boundaries

This is not authentication yet. Step 15 will wire identity/token validation and enforce user permissions at API boundaries.

## Exit Criteria

Step 14 scaffold is complete when:

- User and project domain models exist.
- Repository contracts and memory/MongoDB implementations support users and projects.
- Jobs and queues can carry project ownership.
- Queue admission rejects project mismatches.
- User project submit policy is tested.
