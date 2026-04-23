# Code Review

Review changed files for security and quality issues.

## Instructions

1. Get changed files:
   ```bash
   git diff --name-only HEAD
   git diff --cached --name-only
   ```

2. Read each changed file and check for:

   **CRITICAL (blocks commit):**
   - Hardcoded secrets
   - SQL injection / XSS / path traversal
   - Missing input validation
   - Missing auth on protected routes

   **HIGH (should fix):**
   - Functions >50 lines
   - Files >400 lines
   - Nesting >4 levels
   - Missing error handling
   - console.log statements
   - `any` types

   **MEDIUM (consider):**
   - Missing tests for new code
   - Magic numbers
   - Inconsistent naming

3. Report grouped by severity. For each issue: file, line, problem, fix.

4. Block commit if any CRITICAL issues found.
