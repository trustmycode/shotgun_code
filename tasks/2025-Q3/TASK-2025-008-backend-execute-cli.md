id: TASK-2025-008
title: "Задача 1.4: Обновление логики вызова CLI"
status: done
priority: medium
type: feature
estimate: 1h
assignee: "@unassigned"
created: 2025-09-03
updated: 2025-09-04
parents: [TASK-2025-004]
arch_refs: [ARCH-BACKEND-CLI-MANAGER]
audit_log:
  - {
      date: 2025-09-03,
      user: "@AI-DocArchitect",
      action: "created with status backlog",
    }
  - {
      date: 2025-09-04,
      user: "@Robotic-SSE",
      action: "implemented ExecuteCliTool; status changed to done",
    }
---

## Описание

Адаптировать существующую функцию `ExecuteCliTool` для работы с динамически определенным путем к `cursor-agent`.

## Критерии приемки

- Сигнатура функции изменена на `ExecuteCliTool(prompt string, projectRoot string, executorPath string)`.
- Функция корректно вызывает `cursor-agent` по переданному абсолютному пути.
- На Windows пути к проекту корректно транслируются в формат WSL.

## Definition of Done

- Функция `ExecuteCliTool` успешно вызывает `cursor-agent` по его пути установки.
