---
id: TASK-2025-013
title: "Задача 2.3: Создание потокового чат-компонента (Step3Chat.vue)"
status: backlog
priority: high
type: feature
estimate: 5h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-03
parents: [TASK-2025-010]
arch_refs: [ARCH-FRONTEND-STREAMING-CHAT]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
---
## Описание
Реализовать полнофункциональный UI для чата на Шаге 3, который будет взаимодействовать с бэкенд-прокси для Gemini.

## Критерии приемки
- UI отображает историю сообщений и потоковый ответ от модели в реальном времени.
- Компонент подписывается на события Wails (`newChunk`, `streamEnd`, `streamError`) и корректно их обрабатывает.
- Реализован UI для управления параметром `thinkingBudget` с двумя режимами: "Динамический" и "Ручной".
- Пользователь может скопировать любое сообщение из чата.

## Definition of Done
- Компонент полностью интерактивен, отправляет запросы и отображает потоковые ответы.

