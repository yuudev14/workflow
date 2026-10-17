---
name: write-issue
description: Draft a small, clear GitHub issue with the sections Problem, Expectation and Acceptance criteria. Use when the user says "create an issue", "draft an issue", "write an issue for this", or "issue for this".
---

# write-issue

An issue must be small and easy to read. One issue is one change that fits one PR.

## Steps

1. **Check the size.** If the work needs more than one PR, or both frontend and backend, split it. Write one issue for each PR. The backend issue comes first. If the idea is big, write a short parent issue that only lists the child issue titles.
2. **Write the sections below.** Keep each one to a few lines. Do not write the solution in detail. Do not copy research into the issue. Link the doc instead.
3. **Show the issue in one fenced block** (title first), so the user can copy it. Do not run `gh issue create` unless the user asks you to.
4. When the user asks you to create it, use `gh issue create --repo yuudev14/workflow --title ... --body-file ...`. Do not add labels or assignees.

## Title

Start with a verb. Say the result. Example: `Make the alert dedup fingerprint configurable`.

## Sections

- `## Problem`: What is wrong or missing now. Say what a user or a developer sees. Give one real example. Do not say how to fix it.
- `## Expectation`: What should happen after the change. Say it from the point of view of the user. Keep the "how" out, or put it in one short sentence.
- `## Acceptance criteria`: A checklist (`- [ ]`). One line for each thing that must be true when the work is done. Each line must be easy to test. Include the tests to add. Four to six lines is enough.

Add these only when they help:

- `## Notes`: Facts the developer needs. Links to docs and related issues (`#12`). Known limits. A short idea of the approach, if there is one.
- `## Out of scope`: What this issue does not do. Use it when a reader could expect more.
- `## Open questions`: Choices the user has not made. Keep the list short.

## Writing rules

The text is sent as the user, so follow the "Writing as me" rules in the user's global `CLAUDE.md`: plain words, short sentences, the user's phrases ("Basically", "I noticed", "I believe", "not sure if"), no em dashes, no semicolons. The only list allowed is the checklist in Acceptance criteria.

## Example

```
Title: Make the alert dedup fingerprint configurable

## Problem
I noticed the fingerprint uses six fixed paths in the payload. A source that uses `device.id` has no match, so all its alerts merge into one alert with many occurrences.

## Expectation
The sender can choose the key that is used for dedup. If it does not, the old behavior stays.

## Acceptance criteria
- [ ] The alert payload accepts an optional `dedup_key`, and it replaces the primary entity in the fingerprint
- [ ] Without `dedup_key`, the fingerprint is the same as before
- [ ] A unit test covers the fingerprint with and without `dedup_key`
- [ ] An integration test shows that two alerts with the same `dedup_key` merge

## Out of scope
A config page for the rules. That is a separate frontend issue.
```
