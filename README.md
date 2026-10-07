# Operation HUD for Herdr

Operation HUD shows a small Herdr toast when a meaningful operation finishes in
the repository currently visible in Herdr. It uses Herdr's own notification
toast, the same surface as agent completion notices, so it never takes focus.
No position is passed, so Herdr's toast placement applies (bottom-right unless
`[ui.toast.herdr] position` says otherwise). Git commands launched by
interactive shells and coding agents are detected automatically through Git's
Trace2 event stream.

The toast title names the operation and repository. The body summarizes the
result:

- Success: the new commit for history-writing commands, the branch with
  ahead/behind counts, and the elapsed time.
- Failure: the exit code, the last phase reached (for example pre-commit
  checks), the branch, unresolved conflicts, and the elapsed time.

Read-only probes such as `status`, `diff`, `log`, `rev-parse`, and bare
`branch`/`tag` listings are ignored. So are `add`, which precedes nearly every
commit, and `fetch`, which editors and Lazygit run on timers.

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

Version 0.3 supports macOS and Linux with Git, Herdr 0.9+, and no runtime
dependencies beyond the compiled binary. Raw stdout/stderr is not intercepted;
failure phases come from structured Trace2 events.
