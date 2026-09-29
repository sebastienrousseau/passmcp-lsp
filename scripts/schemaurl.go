// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// schemaurl prints the URL the embedded server.json schema was published
// at, so `make schema-check` fetches the document the code names rather
// than a copy of its URL.
package main

import (
	"fmt"

	"satellion.com/passmcp-lsp/internal/check"
)

func main() { fmt.Println(check.ServerSchemaURL) }
