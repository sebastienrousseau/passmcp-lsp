// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// The VS Code side of passmcp-lsp: start the server over stdio for JSON
// documents, and restart it when its settings change. Everything the
// extension decides is in settings.ts, where it is tested.

import * as vscode from "vscode";
import { LanguageClient, TransportKind } from "vscode-languageclient/node";
import { documentSelector, initializationOptions, launch, Settings } from "./settings";

let client: LanguageClient | undefined;

function readSettings(): Settings {
  const c = vscode.workspace.getConfiguration("passmcpLsp");
  return { path: c.get<string>("path"), passmcpPath: c.get<string>("passmcpPath") };
}

async function start(): Promise<void> {
  const s = readSettings();
  const run = { ...launch(s), transport: TransportKind.stdio };
  client = new LanguageClient(
    "passmcpLsp",
    "passmcp",
    { run, debug: run },
    { documentSelector: documentSelector(), initializationOptions: initializationOptions(s) },
  );
  try {
    await client.start();
  } catch (err) {
    client = undefined;
    void vscode.window.showErrorMessage(
      `passmcp-lsp could not start (${String(err)}). Install it, or set passmcpLsp.path.`,
    );
  }
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(async (e) => {
      if (e.affectsConfiguration("passmcpLsp.path") || e.affectsConfiguration("passmcpLsp.passmcpPath")) {
        await deactivate();
        await start();
      }
    }),
  );
  await start();
}

export async function deactivate(): Promise<void> {
  const c = client;
  client = undefined;
  if (c) {
    await c.stop();
  }
}
