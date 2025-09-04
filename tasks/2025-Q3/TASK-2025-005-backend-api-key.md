---
id: TASK-2025-005
title: "Задача 1.1: Управление API ключом"
status: backlog
priority: high
type: feature
estimate: 1h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-03
parents: [TASK-2025-004]
arch_refs: [ARCH-BACKEND-GEMINI-PROXY]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
---
## Описание
Реализовать безопасное сохранение и загрузку Google AI API ключа пользователя. Ключ должен храниться в файле конфигурации в домашней директории пользователя и не должен передаваться на фронтенд.

## Критерии приемки
- В `app.go` созданы публичные функции `SaveApiKey(key string)` и `LoadApiKey()`.
- Функции корректно сохраняют и читают ключ из файла `~/.shotgun_code/config.json`.
- Ключ, сохраненный через `SaveApiKey`, успешно считывается через `LoadApiKey`.

## Definition of Done
- Функциональность реализована и покрыта юнит-тестами (если применимо).

