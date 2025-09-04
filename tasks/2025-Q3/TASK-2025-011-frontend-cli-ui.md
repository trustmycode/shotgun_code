---
id: TASK-2025-011
title: "Задача 2.1: Реализация условного UI на Шаге 4"
status: backlog
priority: high
type: feature
estimate: 3h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-03
parents: [TASK-2025-010]
arch_refs: [ARCH-FRONTEND-CLI-UI]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
---
## Описание
На Шаге 4 реализовать UI, который будет динамически изменяться в зависимости от статуса установки Cursor CLI и наличия WSL (на Windows).

## Критерии приемки
- При загрузке Шага 4 вызываются бэкенд-функции `CheckPrerequisites` (для WSL) и `CheckCursorInstallation`.
- Если на Windows нет WSL, показывается предупреждение, кнопки неактивны.
- Если `cursor-agent` не установлен, отображается кнопка "Установить Cursor CLI".
- Если `cursor-agent` установлен, отображается кнопка "Выполнить".

## Definition of Done
- UI на Шаге 4 корректно отображает состояние системы и предоставляет соответствующие действия.

