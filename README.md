# Operation HUD for Herdr

Operation HUD opens an animated 90% floating popup while meaningful operations
run in the repository currently visible in Herdr. Git commands launched by
interactive shells and coding agents are detected automatically through Git's
Trace2 event stream.

The initial release deliberately ignores noisy read-only probes such as
`status`, `diff`, `log`, and `rev-parse`. Consecutive operations are displayed
as a short timeline and the popup closes automatically after completion.

## Local development

```sh
go build -o bin/herdr-git-hud ./cmd/herdr-git-hud
herdr plugin link "$PWD" --enabled
herdr plugin action invoke purehate.git-hud.enable
```

Inspect the installation with:

```sh
herdr plugin action invoke purehate.git-hud.status
herdr plugin log list --plugin purehate.git-hud
```

Disable the Git Trace2 target and stop the listener with:

```sh
herdr plugin action invoke purehate.git-hud.disable
```

Other commands can opt in through the wrapper while keeping their original
terminal output and exit code:

```sh
herdr-hud run --label "PR checks" -- gh pr checks --watch
```

The local `odoo-test` helper uses this wrapper automatically. A missing HUD
socket never prevents the wrapped command from running.

## Scope

Version 0.2 supports macOS and Linux with Git, Herdr 0.9+, and no runtime
dependencies beyond the compiled binary. Raw stdout/stderr transfer progress is
not intercepted; the HUD derives phases from structured Trace2 events.
