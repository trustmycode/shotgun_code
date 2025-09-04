---
id: TASK-2025-003
title: "Задача 0.2: Очистка проекта от встроенных бинарных файлов"
status: backlog
priority: medium
type: tech_debt
estimate: 0.5h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-03
parents: [TASK-2025-001]
arch_refs: [ARCH-BACKEND-CLI-MANAGER]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
---
## Описание
Удалить из проекта устаревшую директорию `/bin` с бинарными файлами `cursor-agent` и соответствующую конфигурацию в `wails.json`, чтобы перейти на модель установки по требованию.

## Критерии приемки
- Директория `/bin` в корне проекта удалена.
- В файле `wails.json` ключ `"assetdir"` отсутствует или закомментирован.
- Проект успешно собирается без встроенных ассетов.

## Definition of Done
- Проект больше не содержит и не полагается на встроенные бинарные файлы.

