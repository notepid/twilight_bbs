package db

type migration struct {
	name string
	sql  string
}

var migrations = []migration{
	{
		name: "create users table",
		sql: `
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL COLLATE NOCASE,
				password_hash TEXT NOT NULL,
				real_name TEXT DEFAULT '',
				location TEXT DEFAULT '',
				email TEXT DEFAULT '',
				security_level INTEGER DEFAULT 10,
				total_calls INTEGER DEFAULT 0,
				last_call_at DATETIME,
				ansi_enabled BOOLEAN DEFAULT 1,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`,
	},
	{
		name: "create message areas table",
		sql: `
			CREATE TABLE IF NOT EXISTS message_areas (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				description TEXT DEFAULT '',
				read_level INTEGER DEFAULT 10,
				write_level INTEGER DEFAULT 10,
				sort_order INTEGER DEFAULT 0
			)
		`,
	},
	{
		name: "create messages table",
		sql: `
			CREATE TABLE IF NOT EXISTS messages (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				area_id INTEGER NOT NULL REFERENCES message_areas(id) ON DELETE RESTRICT,
				from_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
				to_user_id INTEGER REFERENCES users(id) ON DELETE RESTRICT,
				subject TEXT NOT NULL,
				body TEXT NOT NULL,
				reply_to_id INTEGER REFERENCES messages(id) ON DELETE RESTRICT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_messages_area ON messages(area_id, id);
			CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_user_id);
		`,
	},
	{
		name: "create message read tracking",
		sql: `
			CREATE TABLE IF NOT EXISTS message_read (
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
				area_id INTEGER NOT NULL REFERENCES message_areas(id) ON DELETE RESTRICT,
				last_read_id INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (user_id, area_id)
			)
		`,
	},
	{
		name: "create file areas table",
		sql: `
			CREATE TABLE IF NOT EXISTS file_areas (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				description TEXT DEFAULT '',
				disk_path TEXT NOT NULL,
				download_level INTEGER DEFAULT 10,
				upload_level INTEGER DEFAULT 20,
				sort_order INTEGER DEFAULT 0
			)
		`,
	},
	{
		name: "create file entries table",
		sql: `
			CREATE TABLE IF NOT EXISTS file_entries (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				area_id INTEGER NOT NULL REFERENCES file_areas(id) ON DELETE RESTRICT,
				filename TEXT NOT NULL,
				description TEXT DEFAULT '',
				size_bytes INTEGER DEFAULT 0,
				uploader_id INTEGER REFERENCES users(id) ON DELETE RESTRICT,
				download_count INTEGER DEFAULT 0,
				uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_file_entries_area ON file_entries(area_id);
		`,
	},
	{
		name: "seed default message areas",
		sql: `
			INSERT OR IGNORE INTO message_areas (id, name, description, sort_order) VALUES
				(1, 'General Discussion', 'General chat and discussion', 1),
				(2, 'Sysop Announcements', 'News from the sysop', 2),
				(3, 'Tech Talk', 'Technology discussion', 3)
		`,
	},
}
