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
- [x] `rulesConfig` как конфиг, а не хардкод (§7, §22)

## Фаза 2 — Время

- [x] Модель тика (1 тик = 1 сек), команда `tick` продвигает `state.gameTime.tick` на `ticksPassed` без ограничения по сменам (§6)
- [x] Покупки/merge/order-команды разрешены в любой момент, без фазовой блокировки (§7)

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

- [x] `get_game_time`, `get_level_view` (viewport-based view-model), `get_workers`, `get_storage_state`, `get_available_orders`, `get_active_orders`, `get_resources`, `get_player_summary` (§37)

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

## Фаза 18 — UI-доработки по фидбеку

- [x] Кнопка найма работника в HUD (`render.HireWorkerButton`), клик обрабатывается в `internal/app` (`hireWorker()`), цена/уровень берутся из Lua (`get_workers.nextPurchasableWorkerLevel/nextPurchaseCost`, посчитано через `config/balance.lua`) — Go не дублирует формулу стоимости
- [x] Запуск игры в полноэкранном режиме (`ebiten.SetFullscreen(true)` в `cmd/mining-game/main.go`)
- [x] Go-тест `TestHireWorkerBuysNextPurchasableLevel`, визуально проверено скриншотом запущенного приложения

## Фаза 19 — Правки по фидбеку: видимость работников и масштаб UI

- [x] Купленные (idle) работники не имеют `positionCellId`, пока не назначены на клетку, поэтому не рисовались на карте — добавлена панель `Workers:` в HUD (`render/ui_renderer.go: drawWorkersPanel`), перечисляющая весь пул работников с id/уровнем/статусом
- [x] `Game.Layout()` теперь возвращает фиксированное логическое разрешение (`render.ScreenWidth/Height = 1280x720`) вместо `outsideWidth/outsideHeight` — Ebitengine сам масштабирует канвас под реальный экран (подтверждено логированием `outsideWidth/outsideHeight≈1511x949` в фуллскрине и сравнением границ отрисованного контента на скриншоте)
- [x] Цвет тумана войны (`unknown`) сделан чуть светлее (`{20,20,24}` → `{38,38,44}`), чтобы отрисованный на весь экран туман не выглядел как пустой чёрный фон
- [x] Визуально проверено: скриншот с предзаполненным сохранением показывает `Workers: worker_1 Lv1 idle` в HUD

## Фаза 20 — Заказы v2: периодическое поступление, цена за единицу, отгрузка частями

Цель: заказы поступают раз в 100 тиков с некоторой вероятностью; у каждого ресурса в заказе своя цена за единицу (из диапазона); заказ можно принять/отклонить; активных заказов может быть несколько; отгрузка идёт частями каждые 50 тиков с оплатой за каждую часть; при нескольких заказах ресурсы распределяются пропорционально; активные заказы и прогресс по ним видны в UI.

Анализ текущего состояния: заказы уже есть (`lua/simulation/orders.lua`), но семантика другая — пул из 3 доступных пополняется мгновенно (нужно поступление по таймеру с вероятностью), оплата единой суммой `rewardMoney` при завершении (нужна цена за единицу и оплата за отгруженные части), распределение выполняется каждый тик (нужно раз в 50), режим по умолчанию `priority_based` (нужен `proportional` с добором остатка, т.к. `floor` доли оставляет неразданный остаток), UI заказов нет.

- [x] Lua-конфиг (`lua/config/rules.lua`, открытые решения — конфиг, не хардкод): `orderAllocationMode = "proportional"`, `orderArrivalIntervalTicks = 100`, `orderArrivalChance = 0.5`, `orderShipmentIntervalTicks = 50`, `maxAvailableOrders = 3`, `orderLifetimeTicks = 1500`
- [x] Генерация заказа: `pricePerUnit` на каждое требование — детерминированный RNG в диапазоне `[basePrice, basePrice*3]`; `rewardMoney` = сумма `requiredAmount * pricePerUnit` (полная стоимость, для UI)
- [x] Отгрузка/оплата: единый помощник `ship()` — списывает со склада, увеличивает `deliveredAmount`, начисляет `withdrawn * pricePerUnit` в `state.money`, событие `order_shipped {orderId, resourceId, amount, payment}`
- [x] `settle(state)` каждый тик: экспирация просроченных; на `tick % 50 == 0` — распределение по accepted-заказам (proportional по умолчанию + добор остатка по приоритету), оплата, завершение полностью отгруженных; на `tick % 100 == 0` — с вероятностью `orderArrivalChance` (RNG от seedPhrase+тика — детерминированно) новый заказ, если доступных меньше `maxAvailableOrders` (`order_arrived`)
- [x] Убрать мгновенный `replenish` из accept/decline/complete; начальное заполнение пула — только в `new_game`; accept при полном наличии ресурсов по-прежнему завершает заказ сразу (правило AGENTS.md), частичной отгрузки при принятии нет — её делает 50-тиковый цикл
- [x] `state.load_state`: бэкфилл новых ключей `rulesConfig` и `pricePerUnit` (из `basePrice`) для старых сейвов
- [x] `read.lua`: детерминированная сортировка списков заказов + текущий `tick` в ответе (для «expires in» в UI)
- [x] Персистентность: колонка `price_per_unit` в `order_requirements` (idempotent `ALTER TABLE ADD COLUMN` в `Open()` для существующих баз), сохранение/чтение в `convert.go`/`load.go`
- [x] UI: панель заказов (доступные — требования, цены, награда, кнопки Accept/Decline; активные — прогресс `delivered/required` по каждому ресурсу); прямоугольники кнопок экспортируются из `render` (как `HireWorkerButton`); хит-тест в `internal/app` до обработки драга; лог `order_shipped`/`order_arrived`/`order_expired` в `handleLuaEvents`
- [x] Тесты: decline не пополняет пул мгновенно; accept при полном складе — мгновенное завершение с оплатой; отгрузка только на границе 50 тиков с оплатой `amount * pricePerUnit`; пропорциональное распределение между двумя заказами; поступление новых заказов на границах 100 тиков (детерминированно от seed); Go-раундтрип сохраняет `pricePerUnit`
- [x] Документация: AGENTS.md §Orders и REQUIREMENTS.md §25–26 — цена за единицу, оплата частями, каденции 50/100 тиков, proportional по умолчанию (заодно заменить устаревшее «в конце смены»)
- [x] `make check` зелёный, коммит после прохождения проверок

## Открытые решения (не блокируют разработку, но требуют явного флага/конфига)

- Переназначение работников во время активной смены
- Полная блокировка активной смены vs аварийные вмешательства
- Формулы: скорость работников, объём компонентов, награды за заказы, стоимость работников/складов
- Лимиты размера пустых пещер
- Выбор Lua VM для Go
- Формат миграций SQLite
- Архитектура UI поверх Ebitengine (чистый Ebitengine vs отдельная UI-система)
