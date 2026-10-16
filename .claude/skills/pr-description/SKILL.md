---
name: pr-description
description: Write the pull request description for the current branch in six sections (Problem, Solution, What we added, What we removed, Test plan, Note) in ASD-STE100 Simplified Technical English. Use when the user says "PR description", "write the PR", "describe this branch", or "add this to the PR".
---

# pr-description

This skill is for this project only. The user asked for Simplified Technical English here. This rule replaces the personal voice rules for PR text.

## Steps

1. Run `git rev-parse --abbrev-ref HEAD`, `git log main..HEAD --no-merges --format=%s`, `git diff --stat main...HEAD` and `git status --short`. Use `origin/main` if it exists.
2. Read the diff only when the commits and the file list do not make the change clear.
3. Write the six sections below. Skip a section only when it is empty, except Problem and Solution.
4. Show the text in one fenced block, so the user can copy it.
5. Add `Fixes #<n>` on its own line if the branch name has an issue number. If the issue needs a backend PR and a frontend PR, use `Refs #<n>` in the backend PR and `Fixes #<n>` in the frontend PR. Name the other PR in the text.
6. If the change alters the UI, add this line after the last section: "Add a before and after screenshot."

## Sections

- `## Problem`: What was wrong. Say what a user or an operator saw. Do not say how you fixed it.
- `## Solution`: The fix in two to four sentences. Say what changed, not how the code works.
- `## What we added`: A list. One item for each feature, behavior, setting or migration. Say what it does.
- `## What we removed`: A list. Name each service, package, setting or behavior that is gone. Say what replaced it.
- `## Test plan`: A list. Say what you tested and the result. Say what you did not test.
- `## Note`: A list of what the reviewer must do or know. Include the commands to run (migrations, settings) and the known limits.

## Writing rules (ASD-STE100)

- Write short sentences. Use 20 words or fewer for an instruction and 25 or fewer for a description.
- Say one idea in each sentence.
- Use active voice and simple tenses: the simple present, the simple past or the simple future.
- Use one word for one meaning. Do not use idioms, jokes or figures of speech.
- Do not use noun clusters of more than three words.
- Use plain words. Write "use", not "utilize". Write "make sure", not "ensure".
- Keep the same name for the same thing in the whole text.
- Do not use code blocks, file paths or function names, unless the reviewer must type or search for them.
- Give numbers with their unit and their condition (for example, "2,000 runs in 13 seconds on a laptop").

## Limits

- Do not run `gh pr create` or `gh pr edit` unless the user asks for it in this turn. Do not push the branch without a request.
- Do not mention Claude.
