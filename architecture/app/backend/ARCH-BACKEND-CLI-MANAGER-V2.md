---
id: ARCH-BACKEND-CLI-MANAGER
title: "Бэкенд: Менеджер CLI v2 (Codex CLI)"
type: component
layer: application
owner: '@AI-DocArchitect'
version: v2
status: planned
created: 2025-09-04
updated: 2025-09-04
tags: [backend, cli, codex-cli]
depends_on: []
referenced_by: []
---
## Контекст
Этот компонент отвечает за управление взаимодействием с внешним инструментом командной строки на стороне бэкенда. В версии 2 планируется переход с `cursor-agent` на `codex-cli`.

## Структура
Основные изменения коснутся функций в файле `app.go`.
- `CheckCursorInstallation` → `CheckCodexCli`
- `InstallCursorCli` → `InstallCodexCli`
- `ExecuteCliTool` → `ExecuteCodexCli`
- Новая функция: `AuthorizeCodexCli`

## Поведение
- **`CheckCodexCli`**: Будет проверять наличие `codex` в `PATH` и валидность конфигурационного файла `~/.codex/config.toml`. Функция должна возвращать статус: `not_installed`, `installed_not_authed`, `installed_and_authed`.
- **`InstallCodexCli`**: Будет выполнять команду `npm install -g @openai/codex`.
- **`AuthorizeCodexCli`**: Будет реализовывать кросс-платформенный запуск нового окна терминала с командой `codex` для интерактивной авторизации пользователя.
- **`ExecuteCodexCli`**: Будет вызывать `codex exec "<prompt>"` с установкой правильного рабочего каталога (`cmd.Dir`) в корень проекта пользователя.

## Эволюция
### Планируется
- Заменить всю логику, связанную с `cursor-agent`, на эквивалентную для `codex-cli`.
- Реализовать механизм запуска внешнего терминала для авторизации.

### Историческая
- v2: Планирование перехода на `codex-cli`.
