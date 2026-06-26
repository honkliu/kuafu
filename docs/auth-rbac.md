# Auth And RBAC

Status: Step 15 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu needs API-boundary identity before project-scoped admission can be trusted.

## Current Scaffold

- `internal/auth` defines an `Authenticator` interface.
- `HeaderAuthenticator` resolves `X-Kuafu-User` against the repository user store.
- `AnonymousAuthenticator` keeps lab compatibility for requests without project-scoped authorization.
- Job submission with `project` now authenticates the caller and calls `policy.CanSubmit`.
- Lab mode seeds `lab-admin`, `lab-user`, and `lab-viewer` and wires `HeaderAuthenticator` into the runnable API server.

## Security Boundary

The header authenticator is a development scaffold. Production must replace it with OIDC/JWT validation before accepting external traffic.

Lab project-scoped submission example:

```powershell
curl -H "X-Kuafu-User: lab-user" `
  -H "Content-Type: application/json" `
  -d '{"name":"hello","project":"lab","queue":"default","command":"echo hello","gpuCount":1}' `
  http://localhost:8080/api/v1/jobs
```

Production mode still fails fast, so this scaffold does not expose a fake production auth system.

## Exit Criteria

Step 15 scaffold is complete when:

- API has an authenticator interface.
- Project-scoped job submission derives user identity from the auth layer.
- Viewers/non-submitters are rejected.
- Submitters/admins are allowed.
- Docs clearly state header auth is not production OIDC.
