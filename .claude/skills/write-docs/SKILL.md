---
name: write-docs
description: Write a doc page in docs/ that explains how a part of YTSoar works (auth, connectors, database, conditions and so on), in plain English, with Mermaid diagrams. Use when the user says "create a doc", "write docs for", "document how X works", "explain X in a doc", "add a diagram", or "make it easier to understand".
---

# write-docs

This skill is for this project. The reader is new to the codebase. They know React, JavaScript and Python. English is not their first language. The doc must be easy to scan and easy to understand.

## Steps

1. **Read the real code first.** Use Grep and Read on `app/ytsoar`, `app/connectors`, `app/frontend`, the migrations in `app/ytsoar/db/migrations` and the tests. Never write from memory. When a claim is risky (a default, a limit, a difference between two tools), run a small check to confirm it.
2. **One doc for each topic, in `docs/<topic>.md`.** Use a kebab-case name. If a doc for the topic exists (`worker-guide.md`, `river-guide.md`, `runtime-resolver.md`, `auth.md`, `database.md`, and so on), update it. Do not add a new file when an existing one fits.
3. **Use this structure.**
   - Title and 2 or 3 sentences: what the page explains.
   - **The big picture:** a small table of the key words, then one diagram.
   - Numbered sections in the order things happen. Use real example data.
   - Tables for rules, options, routes and errors.
   - **Limits and gotchas** at the end. Say what is not built yet.
   - Links to related docs.
4. **Write plain English.** Short sentences. Active voice. One idea in each sentence. Explain each term once, when it first appears. No marketing words. No em dashes.
5. **Do not list files by default.** Name a file, a function or a setting only when the reader needs it to do something (run a command, change a setting, find where to edit). Do not add "Where the code is" tables. Do not use line numbers.
6. **Draw diagrams in Mermaid.**
   - `flowchart` for decisions and data flow, `sequenceDiagram` for who calls whom, `erDiagram` for tables.
   - At most about 12 nodes in a diagram. Split a big one. Label every arrow.
   - Put labels with special characters in quotes. Use `<br/>` for a new line.
   - Check that each diagram renders: save it to a `.mmd` file and run `npx --yes @mermaid-js/mermaid-cli -p pp.json -i x.mmd -o x.svg`, where `pp.json` is `{"args":["--no-sandbox"]}`. Fix every error.
7. **Verify the facts.** Check each name, number and default against the code.
8. **Link the doc** from the "Other docs" list in `README.md`.
9. Do not commit. Do not create GitHub issues (draft them in chat). In the chat reply, follow the Reporting rule in `CLAUDE.md` (Simplified Technical English). Tell the user the path of the doc and give a short summary.
