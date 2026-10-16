package web

import "embed"

//go:embed index.html manage.js manage.css my-vault.svg restore.html restore.js
var Files embed.FS
