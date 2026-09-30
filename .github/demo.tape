# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The README demo: the Quick Start, run. `passmcp-lsp check` flags a
# policy with a misspelled rule, then passes it once the key is corrected.
# Rendered by `make demo`, which builds the binary and runs this in
# build/demo/work, so the policy it writes is thrown away.
Output .github/demo.gif
Require passmcp-lsp
Set Shell "bash"
Set FontSize 13
Set Width 1400
Set Height 210
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 35ms

Hide
Type "PS1='$ '; cd build/demo/work; clear" Enter
Show

Type `printf '{"version": 1, "name": "platform", "must_pas": ["net.tls"]}\n' > company.policy.json` Sleep 400ms Enter
Sleep 500ms
Type "passmcp-lsp check company.policy.json" Sleep 400ms Enter
Sleep 2s
Type `printf '{"version": 1, "name": "platform", "must_pass": ["net.tls"]}\n' > company.policy.json` Sleep 400ms Enter
Sleep 500ms
Type "passmcp-lsp check company.policy.json" Sleep 400ms Enter
Sleep 4s
