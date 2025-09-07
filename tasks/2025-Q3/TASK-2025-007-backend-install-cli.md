---
id: TASK-2025-007
title: "Задача 1.3: Реализация установщика Cursor CLI"
status: done
priority: high
type: feature
estimate: 2h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-04
parents: [TASK-2025-004]
arch_refs: [ARCH-BACKEND-CLI-MANAGER]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
  - {date: 2025-09-04, user: "@Robotic-SSE", action: "implemented InstallCursorCli; status changed to done"}
---
## Описание
Создать функцию, которая безопасно скачивает и выполняет официальный установочный скрипт Cursor CLI. Процесс должен быть безопасным и очищать временные файлы после завершения.

## Критерии приемки
- Создана публичная функция `InstallCursorCli() error`.
- Функция скачивает скрипт с `https://cursor.com/install` по HTTPS.
- Скрипт сохраняется во временный файл.
- Скрипт выполняется нативно на macOS/Linux и через WSL на Windows.
- Временный файл гарантированно удаляется после выполнения.

## Definition of Done
- Функция успешно устанавливает `cursor-agent` на всех целевых платформах.

