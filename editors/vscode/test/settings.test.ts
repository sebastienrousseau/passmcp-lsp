// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test } from "node:test";
import * as assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { defaultServer, documentSelector, initializationOptions, launch } from "../src/settings";

test("launch defaults to passmcp-lsp on PATH", () => {
  assert.deepEqual(launch({}), { command: defaultServer, args: [] });
  assert.deepEqual(launch({ path: "   " }), { command: "passmcp-lsp", args: [] });
});

test("launch uses the configured path, trimmed, with no extra arguments", () => {
  assert.deepEqual(launch({ path: " /opt/bin/passmcp-lsp " }), { command: "/opt/bin/passmcp-lsp", args: [] });
});

test("initializationOptions carries passmcpPath only when it is set", () => {
  assert.deepEqual(initializationOptions({}), {});
  assert.deepEqual(initializationOptions({ passmcpPath: "" }), {});
  assert.deepEqual(initializationOptions({ passmcpPath: " /usr/local/bin/passmcp" }), {
    passmcpPath: "/usr/local/bin/passmcp",
  });
});

test("documentSelector covers JSON and JSON with comments, saved or not", () => {
  const sel = documentSelector();
  assert.equal(sel.length, 4);
  for (const scheme of ["file", "untitled"]) {
    for (const language of ["json", "jsonc"]) {
      assert.ok(sel.some((f) => f.scheme === scheme && f.language === language), `${scheme}/${language}`);
    }
  }
});

test("package.json activates on the languages the selector covers", () => {
  const manifest = JSON.parse(readFileSync(join(__dirname, "..", "..", "package.json"), "utf8")) as {
    activationEvents: string[];
  };
  const languages = new Set(documentSelector().map((f) => f.language));
  for (const language of languages) {
    assert.ok(manifest.activationEvents.includes(`onLanguage:${language}`), language);
  }
});
