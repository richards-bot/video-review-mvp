# Plan

Create an implementation plan before writing any code.

## Instructions

1. **Restate requirements**
   - Prefer OpenSpec change artifacts (`openspec/changes/...`) when present.
   - Else use `openspec/specs/...`.
   - If neither exists, propose creating one first.

2. **Produce a phased plan** with explicit verification and checkpoint gates:
   ```
   Phase 1: [Name]
   - Task -> verify: [how to check]
   - Checkpoint gate -> user approves before Phase 2
   ```

3. **Include mandatory sections**
   - Assumptions
   - Non-goals
   - Risks (with mitigations)
   - Open questions that block coding

4. **Testing strategy (outside-in)**
   - Acceptance/E2E first
   - Integration next
   - Unit tests for complex logic

5. **WAIT FOR CONFIRMATION**
   Do NOT write code until the user explicitly approves the plan.

After approval:
1. `/start-bead`
2. Implement phase by phase
3. `/checkpoint` at each phase gate (human review)
4. `/complete-bead` when done
