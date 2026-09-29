// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// What the extension derives from its settings, kept free of the vscode
// module so that it can be tested with Node alone.

/** The extension's settings, as read from the passmcpLsp section. */
export interface Settings {
  path?: string;
  passmcpPath?: string;
}

/** The program to start and its arguments. */
export interface Launch {
  command: string;
  args: string[];
}

/** The server's program name when the setting is empty. */
export const defaultServer = "passmcp-lsp";

/**
 * The server to start. The language client adds --stdio itself for a stdio
 * transport, so no arguments are needed here.
 */
export function launch(s: Settings): Launch {
  const command = (s.path ?? "").trim();
  return { command: command === "" ? defaultServer : command, args: [] };
}

/**
 * The options sent with initialize: the passmcp program to ask for
 * guidance, when the user named one.
 */
export function initializationOptions(s: Settings): { passmcpPath?: string } {
  const passmcpPath = (s.passmcpPath ?? "").trim();
  return passmcpPath === "" ? {} : { passmcpPath };
}

/** A document filter, in the shape the language client takes. */
export interface DocumentFilter {
  scheme: string;
  language: string;
}

/**
 * Every JSON document, saved or not. The server recognises the MCP
 * artefacts among them, by name and by content, and stays silent on the
 * rest, so a narrower selector would only hide policies and attestations
 * whose file names follow no convention.
 */
export function documentSelector(): DocumentFilter[] {
  const out: DocumentFilter[] = [];
  for (const scheme of ["file", "untitled"]) {
    for (const language of ["json", "jsonc"]) {
      out.push({ scheme, language });
    }
  }
  return out;
}
