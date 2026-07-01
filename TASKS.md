# TASKS.md

План работ по реализации Idle Mining Game (Go + Ebitengine + Lua), составлен на основе `REQUIREMENTS.md` / `AGENTS.md`.

Порядок фаз важен: Lua-движок должен быть тестируемым и рабочим независимо от Ebitengine до подключения рендера и персистентности.

## Фаза 0 — Скелет проекта

- [x] Инициализировать `go.mod`, структуру каталогов из REQUIREMENTS.md §31 (`cmd/mining-game`, `internal/app|render|luaengine|persistence|application`, `lua/...`, `docs/...`)
- [x] Выбрать и подключить Lua VM для Go (открытое решение REQUIREMENTS.md §43.4, напр. gopher-lua)
- [x] Настроить `Makefile` с целью `check` (go fmt/vet/test/-race/build, lua tests, stylua)
- [x] Создать заглушки `docs/architecture.md`, `docs/game-design.md`, `docs/engine-api.md`, `docs/persistence.md`

## Фаза 1 — Lua engine: каркас API

- [x] `engine.apply(command)` / `engine.read(query)` / `engine.export_state()` / `engine.load_state(state)` (§33)
- [x] Формат результата `apply`: `{ok, events, patch}` / `{ok=false, error={code, message, details}}` (§38)
- [x] Диспетчер команд/запросов по `type`, валидация в Lua, без host-функций с игровой логикой (§5)
- [x] `rulesConfig` (в т.ч. `allowWorkerReassignmentDuringShift`) как конфиг, а не хардкод (§7, §22)

## Фаза 2 — Время и фазы игры

- [x] Модель тика (1 тик = 1 сек), команда `tick` с ограничением одной сменой за вызов, возврат `processedTicks`/`remainingTicks` (§6)
- [x] `fast_forward_to_shift_end`, `start_next_shift`
- [x] Фазы `shift_running` / `shift_planning`, блокировка purchase/merge/order-команд вне planning (§7)
- [x] Событие `shift_completed` → `autosave_requested` → `shift_planning_started`

## Фаза 3 — Генерация карты

- [x] Модель чанка 32×32, детерминированная генерация по `(seedPhrase, levelDepth, chunkX, chunkY, generatorVersion)` (§8)
- [x] Генерация стартовой области 5×5 чанков, центральный чанк со стартовой зоной 3×3 (§9)
- [x] Размещение зоны спуска 3×3 в нецентральном чанке, скрытой до разведки (§9)
- [x] Типы клеток: `empty`, `deposit`, `obstacle`, `stairs_area` (§10)
- [x] Валидация связности: путь от входа до спуска, ширина коридоров ≤3, детерминированный retry/фикс при провале (§13, §14)
- [x] Ограничение размера больших пустых пещер

## Фаза 4 — Видимость и достижимость

- [x] Разделение `visibility` (`unknown`/`scouted`) и `accessibility` (`unreachable`/`reachable`) (§11)
- [x] Радиус разведки 5 клеток от каждой достижимой/открытой клетки
- [x] Flood fill связных пустых областей — мгновенное открытие всей области (§12)
- [x] Догенерация соседних чанков при выходе видимости/flood fill за границу

## Фаза 5 — Клетки и добыча

- [x] Модель `CellComponent` (`rock`/`resource`, `initialAmount`, `remainingAmount`, `ratio`) (§15)
- [x] Пропорциональная выработка компонентов за тик; `rock` не идёт на склад
- [x] Блокировка добычи ресурса при заполненном складе без потери ресурса, статус `blocked_by_storage` (§16)
- [x] Переход клетки `deposit` → `empty` при полной выработке всех компонентов

## Фаза 6 — Работники

- [x] Модель `Worker` (id, level, speed, state, assignedLevelId, targetCellId, positionCellId, assignmentMode) (§17)
- [x] Команда `assign_worker_to_target_cell` + полная валидация (доступность цели, соседство, достижимость позиции, занятость, состояние работника, невыработанность клетки) (§21)
- [x] До 4 работников на клетку (по одной стороне каждый), суммирование скоростей, без учёта индивидуального вклада (§20)
- [x] `AssignmentMode`: `shift_task` / `until_completed` (§22)
- [x] `stop_worker`
- [x] Merge 2→1 только для свободных работников (§18)
- [x] Покупка работников: `maxPurchasableWorkerLevel = max(1, highestUnlockedWorkerLevel - 2)`, только в planning (§19)

## Фаза 7 — Склады

- [x] Модель `Storage` (id, resourceId, level, capacity, storedAmount), суммирование ёмкости по ресурсу (§23)
- [x] `buy_storage`, `upgrade_storage` — только в planning

## Фаза 8 — Деньги и заказы

- [x] Модель `Order`/`order_requirements`, состояния `available/accepted/completed/expired/declined` (§25)
- [x] `accept_order`, `decline_order`, `set_order_priority`, `complete_order_immediately`
- [x] Немедленное закрытие заказа при достаточных ресурсах; распределение в конце смены иначе
- [x] `OrderAllocationMode`: `priority_based` (MVP) и `proportional` (§26)

## Фаза 9 — Ресурсы и глубина

- [x] `ResourceConfig` (id, rarity, unlockDepth, basePrice, storageBaseCapacity, generationWeightByDepth), лимит 12 ресурсов (§27)
- [x] Гарантия минимум одного нового ресурса на новый уровень до открытия всех 12

## Фаза 10 — Уровни

- [x] `create_next_level` — создание следующего уровня по открытии пути к зоне спуска

## Фаза 11 — Запросы (read-only)

- [x] `get_game_phase`, `get_game_time`, `get_level_view` (viewport-based view-model), `get_workers`, `get_storage_state`, `get_available_orders`, `get_active_orders`, `get_resources`, `get_player_summary`, `get_shift_summary` (§37)

## Фаза 12 — Go: Lua runtime и биндинги

- [x] `internal/luaengine`: `runtime.go`, `apply.go`, `read.go`, `state.go`, `bindings.go`
- [x] Маппинг структурированных ошибок Lua (`error.code`) в Go-ошибки, без парсинга `message`

## Фаза 13 — Go: Ebitengine app

- [x] `internal/app`: окно, `Update`/`Draw`, input, camera
- [x] Accumulator для игровых тиков (независимо от FPS Ebitengine) (§34)
- [x] Обработка событий из `apply` (`handleLuaEvents`), включая `autosave_requested`

## Фаза 14 — Go: рендер/UI

- [x] `internal/render`: тайлы карты, работники, UI складов/заказов/планирования смены, sprite/tile atlas
- [x] Рендер только на основе `read`/view-model, без кеширования authoritative-состояния

## Фаза 15 — Персистентность (SQLite)

- [x] Схема таблиц: `saves, levels, chunks, cells, cell_components, workers, storages, orders, order_requirements` (§28, требования AGENTS.md §Required SQLite Structure)
- [x] Миграции (выбрано: idempotent `CREATE TABLE IF NOT EXISTS` при `Open()`, без отдельного фреймворка — открытое решение §43.5 закрыто минимальным вариантом)
- [x] `internal/persistence`: `LoadEngine`, `SaveEngine`, `CreateNewEngine` — только конвертация состояния, без игровой логики (§29)
- [x] Автосейв по событию `autosave_requested` (`internal/app/update.go`), ручное сохранение через `Game.SaveNow()` (§30)

## Фаза 16 — Тесты

- [x] Lua-тесты механик (без Ebitengine, `tests/run.lua`, 18 тестов): генерация по seed, детерминизм, старт/спуск 3×3, путь старт→спуск, видимость 5 клеток, flood fill, пропорциональная добыча, блокировка склада без потери ресурса, выработка клетки, запрет дублирования позиций, merge 2→1, формула покупки уровня, смена 300 тиков, fast-forward до конца смены, немедленное закрытие заказов, событие autosave (§39). Тест на «до 4 работников на клетку» и явный лимит размера пещер не выделены отдельно — покрыты структурно валидацией назначения/генерации.
- [x] Go-тесты: старт Lua runtime и `apply`/`read` (`internal/luaengine`), маппинг ошибок по `code`, сохранение/загрузка SQLite и эквивалентность состояния после restore (`internal/persistence`), autosave-событие вызывает persistence adapter (`internal/app`) (§40)

## Фаза 17 — Финальная проверка перед сдачей

- [x] `go fmt ./... && go vet ./... && go test ./... && go test -race ./... && go build ./...` — все зелёные
- [x] `golangci-lint run` — установлен через Homebrew, 0 замечаний (после исправления 9 errcheck на `Close`/`Rollback`)
- [x] `lua tests/run.lua` / `make test-lua` — 18/18 тестов пройдено
- [x] `stylua --check lua/` — установлен через Homebrew; добавлен `stylua.toml` (2 пробела, ширина строки 120), весь `lua/`+`tests/` отформатирован и проходит чисто
- [x] Обновить `docs/*.md` при изменении архитектуры (`docs/architecture.md`, `docs/persistence.md`)
- [x] Commit только при прохождении всех проверок — создан по запросу пользователя после того, как все проверки прошли (`f039fae`)

## Открытые решения (не блокируют разработку, но требуют явного флага/конфига)

- Переназначение работников во время активной смены
- Полная блокировка активной смены vs аварийные вмешательства
- Формулы: скорость работников, объём компонентов, награды за заказы, стоимость работников/складов
- Лимиты размера пустых пещер
- Выбор Lua VM для Go
- Формат миграций SQLite
- Архитектура UI поверх Ebitengine (чистый Ebitengine vs отдельная UI-система)
