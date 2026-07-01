package persistence

// schema defines the structured save tables (REQUIREMENTS.md §28/§29 - never a
// single JSON blob). Applied idempotently on Open(); a dedicated migration
// framework is an open design decision (REQUIREMENTS.md §43.5) not needed yet
// for a single, additive schema.
const schema = `
CREATE TABLE IF NOT EXISTS saves (
	id TEXT PRIMARY KEY,
	seed_phrase TEXT NOT NULL,
	generator_version INTEGER NOT NULL,
	schema_version INTEGER NOT NULL,
	tick INTEGER NOT NULL,
	shift_index INTEGER NOT NULL,
	phase TEXT NOT NULL,
	money REAL NOT NULL,
	highest_unlocked_worker_level INTEGER NOT NULL,
	next_worker_id INTEGER NOT NULL,
	next_storage_id INTEGER NOT NULL,
	next_order_id INTEGER NOT NULL,
	next_level_id INTEGER NOT NULL,
	allow_worker_reassignment_during_shift INTEGER NOT NULL,
	order_allocation_mode TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS levels (
	id TEXT NOT NULL,
	save_id TEXT NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
	depth INTEGER NOT NULL,
	entrance_x INTEGER NOT NULL,
	entrance_y INTEGER NOT NULL,
	stairs_x INTEGER NOT NULL,
	stairs_y INTEGER NOT NULL,
	stairs_chunk_cx INTEGER NOT NULL,
	stairs_chunk_cy INTEGER NOT NULL,
	stairs_reachable INTEGER NOT NULL,
	next_level_id TEXT,
	PRIMARY KEY (save_id, id)
);

CREATE TABLE IF NOT EXISTS chunks (
	save_id TEXT NOT NULL,
	level_id TEXT NOT NULL,
	cx INTEGER NOT NULL,
	cy INTEGER NOT NULL,
	PRIMARY KEY (save_id, level_id, cx, cy),
	FOREIGN KEY (save_id, level_id) REFERENCES levels(save_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cells (
	save_id TEXT NOT NULL,
	level_id TEXT NOT NULL,
	x INTEGER NOT NULL,
	y INTEGER NOT NULL,
	kind TEXT NOT NULL,
	visibility TEXT NOT NULL,
	accessibility TEXT NOT NULL,
	occupied_by TEXT,
	PRIMARY KEY (save_id, level_id, x, y),
	FOREIGN KEY (save_id, level_id) REFERENCES levels(save_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cell_components (
	save_id TEXT NOT NULL,
	level_id TEXT NOT NULL,
	cell_x INTEGER NOT NULL,
	cell_y INTEGER NOT NULL,
	idx INTEGER NOT NULL,
	type TEXT NOT NULL,
	resource_id TEXT,
	initial_amount REAL NOT NULL,
	remaining_amount REAL NOT NULL,
	ratio REAL NOT NULL,
	PRIMARY KEY (save_id, level_id, cell_x, cell_y, idx),
	FOREIGN KEY (save_id, level_id, cell_x, cell_y) REFERENCES cells(save_id, level_id, x, y) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workers (
	id TEXT NOT NULL,
	save_id TEXT NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
	level INTEGER NOT NULL,
	speed REAL NOT NULL,
	state TEXT NOT NULL,
	assigned_level_id TEXT,
	target_cell_id TEXT,
	position_cell_id TEXT,
	assignment_mode TEXT,
	assigned_side TEXT,
	PRIMARY KEY (save_id, id)
);

CREATE TABLE IF NOT EXISTS storages (
	id TEXT NOT NULL,
	save_id TEXT NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
	resource_id TEXT NOT NULL,
	level INTEGER NOT NULL,
	capacity REAL NOT NULL,
	stored_amount REAL NOT NULL,
	PRIMARY KEY (save_id, id)
);

CREATE TABLE IF NOT EXISTS orders (
	id TEXT NOT NULL,
	save_id TEXT NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
	state TEXT NOT NULL,
	reward_money REAL NOT NULL,
	expires_at_tick INTEGER NOT NULL,
	accepted_at_tick INTEGER,
	priority INTEGER NOT NULL,
	PRIMARY KEY (save_id, id)
);

CREATE TABLE IF NOT EXISTS order_requirements (
	save_id TEXT NOT NULL,
	order_id TEXT NOT NULL,
	idx INTEGER NOT NULL,
	resource_id TEXT NOT NULL,
	required_amount REAL NOT NULL,
	delivered_amount REAL NOT NULL,
	PRIMARY KEY (save_id, order_id, idx),
	FOREIGN KEY (save_id, order_id) REFERENCES orders(save_id, id) ON DELETE CASCADE
);
`
