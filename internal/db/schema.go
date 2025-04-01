package db

import _ "embed"

var Schema string = `
CREATE TABLE IF NOT EXISTS fragments
(
    id   INTEGER PRIMARY KEY,

    label TEXT DEFAULT 'default',

    name TEXT,
    content TEXT,

    embedding_model TEXT,
    embedding_vector BLOB,

    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),

    CONSTRAINT unique_lable_name UNIQUE (label, name)

);`
