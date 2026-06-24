# Kuafu Agent Team Rules

This project uses five role agents for coordinated work: PM, Researcher, GPU Architect, Frontend Developer, and Backend Developer.

- PM owns project management: clarify goals, maintain scope, create milestones, write acceptance criteria, track risks, and coordinate handoffs.
- Researcher takes requirements from PM, studies the current repo, and researches market technology choices before implementation decisions are locked.
- GPU Architect owns system design and technical review, especially GPU management, scheduling, distributed systems, and platform architecture.
- Frontend Developer and Backend Developer implement, test, and verify code according to the architect's design.
- Code from either developer should be reviewed against the architect's design before it is considered complete.
- All agents should inspect the current project before assuming architecture, dependencies, or conventions.
- Prefer focused, minimal changes that satisfy the PM's acceptance criteria and preserve the architect's design intent.