---
slug: static-browser-todo
archetype: static-browser
title: Simple Todo App
---

# Simple Todo App Spec

**Stack**

- Raw HTML
- CSS in a `<style>` tag
- Inline JavaScript in a `<script>` tag
- No framework
- No build step
- No backend
- Optional persistence with `localStorage`

**File Structure**

```text
index.html
```

**Goal**
Build a tiny browser-based todo list app where users can add, complete, edit, delete, and filter tasks.

**Core Features**

1. Add a todo from a text input.
2. Render todos in a list.
3. Mark todos complete/incomplete with a checkbox.
4. Delete a todo.
5. Filter by All, Active, and Completed.
6. Show active todo count.
7. Clear completed todos.
8. Persist todos in `localStorage`.

**Acceptance Criteria**

- App works by opening `index.html` directly in a browser.
- No external dependencies are required.
- Refreshing the page keeps existing todos.
