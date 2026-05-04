# Emoji Tui

A terminal emoji picker that copies the chosen emoji to your clipboard.

## Install

```bash
go install github.com/ConnorsApps/emoji-tui/cmd/emoji@latest
```

Requires Go 1.26+ and `$GOPATH/bin` (typically `$HOME/go/bin`) on your `PATH`.

The installed binary is named `emoji`

## Usage

- `emoji` — open the interactive TUI: fuzzy-search by name or keyword, navigate, press enter to copy.
- `emoji smile happy` — non-interactive: copies the best match for the query and prints it. Whitespace-separated terms are AND-ed across each emoji's name and keywords.

## Notes

- Clipboard support is provided by [`github.com/atotto/clipboard`](https://github.com/atotto/clipboard). On Linux you'll need `xclip` or `xsel` (Wayland: `wl-clipboard`) installed.
- The emoji set is built into the binary — see `cmd/emoji/emojis.go`.
