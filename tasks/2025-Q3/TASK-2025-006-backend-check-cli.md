---
id: TASK-2025-006
title: "Задача 1.2: Реализация проверки установки Cursor CLI"
status: backlog
priority: high
type: feature
estimate: 2h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-03
parents: [TASK-2025-004]
arch_refs: [ARCH-BACKEND-CLI-MANAGER]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
---
## Описание
Создать надежную функцию для определения, установлен ли `cursor-agent` и где он находится. Функция должна работать кросс-платформенно, включая проверку внутри WSL для Windows.

## Критерии приемки
- Создана публичная функция `CheckCursorInstallation() (map[string]string, error)`.
- Функция корректно проверяет наличие файла `~/.cursor/bin/cursor-agent` на macOS/Linux.
- Функция корректно проверяет наличие файла `~/.cursor/bin/cursor-agent` внутри WSL на Windows.
- Функция возвращает карту вида `{"status": "installed", "path": "..."}` или `{"status": "not_installed"}`.

## Definition of Done
- Функция корректно определяет статус установки на всех целевых платформах.

