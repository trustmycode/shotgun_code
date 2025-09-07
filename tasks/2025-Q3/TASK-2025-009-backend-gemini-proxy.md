---
id: TASK-2025-009
title: "Задача 1.5: Реализация потоковой прокси-функции для Google AI"
status: done
priority: high
type: feature
estimate: 2h
assignee: '@unassigned'
created: 2025-09-03
updated: 2025-09-04
parents: [TASK-2025-004]
arch_refs: [ARCH-BACKEND-GEMINI-PROXY]
audit_log:
  - {date: 2025-09-03, user: "@AI-DocArchitect", action: "created with status backlog"}
  - {date: 2025-09-04, user: "@Robotic-SSE", action: "implemented CommunicateWithGoogleAI; status changed to done"}
---
## Описание
Создать на бэкенде функцию, которая будет проксировать запросы к Gemini API в потоковом режиме, управляя API-ключом и транслируя ответы на фронтенд через события Wails.

## Критерии приемки
- Создана публичная функция `CommunicateWithGoogleAI(ctx context.Context, request ChatRequest)`.
- Функция использует `genai` SDK для потокового взаимодействия с моделью `gemini-2.5-pro-latest`.
- Фрагменты ответа транслируются на фронтенд через события Wails (`newChunk`, `streamEnd`, `streamError`).
- Параметры `Temperature` и `ThinkingBudget` из запроса корректно передаются в API.

## Definition of Done
- Функция устанавливает соединение с Google API и начинает транслировать события на фронтенд.

