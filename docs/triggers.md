# Triggers: install and pre-run scripts

Triggers let a package (or a project) run Lua scripts at two points in the
`bz` lifecycle:

| Trigger         | When it runs                                   | Purpose |
|-----------------|------------------------------------------------|---------|
| `installScript` | Once, after the package is downloaded/extracted | Perform one-time installation work (download the real software, unpack it, set permissions) |
| `preRunScript`  | Once per `bz` invocation, before the command runs | Compute environment variables, aliases, and PATH entries dynamically |

Together they enable **meta-packages**: a small `bz` package that contains no
binaries itself, only an install script that fetches the actual software and a
pre-run script that points the environment at it.

## Declaring triggers

In a project's `.bz.hcl`:

```hcl
deps = [
  "github.com/bazurto/openjdk#17"
]

triggers {
  installScript = "$DIR/install.lua"
  preRunScript  = "$DIR/prerun.lua"
}
```

In a package's `.bz.lock` (JSON), which is what published packages ship:

```json
{
  "env": { "META_HOME": "$DIR" },
  "triggers": {
    "installScript": "$DIR/install.lua",
    "preRunScript": "$DIR/prerun.lua"
  }
}
```

Script paths are shell-expanded with the dependency's environment, so `$DIR`
(the extracted package directory), `$BINDIR`, and any exported or implicit
variables are available.

## Install script

Runs when the dependency tree is resolved. Sub-dependency install scripts run
before their parents, so a meta-package can rely on its dependencies being
fully installed.

The script's environment contains the OS environment plus the dependency's
resolved variables (`DIR`, `BINDIR`, `CURDIR`, `BZ_PROJECT_DIR`, `GOOS`,
`GOARCH`, exports, and the implicit `<NAME>_DIR` / `<NAME>_BINDIR` variables).
The package's own `preRunScript` has *not* run yet at this point — the pre-run
script typically inspects what the install script produced.

### Run-once semantics

After a successful run, `bz` writes a `<script>.done` marker next to the
script containing a hash of the script's content. The script is skipped as
long as the marker matches; **editing the script re-triggers installation**.
The marker is only written on success.

Practical consequences:

- **Install scripts must be idempotent.** A script that fails midway leaves
  no marker and will run again in full on the next invocation. Write scripts
  so re-running them is safe (e.g. check with `bz.stat` before downloading).
- If a project declares its own `installScript`, the `.done` marker is
  created inside the project directory — add `*.done` to the project's
  `.gitignore`.
- Deleting a package's cache directory (`~/.bz/cache/deps/...`) also deletes
  the marker, so a re-extracted package re-installs as expected.

### Example: meta-package install script

```lua
-- install.lua: download and unpack the real software next to this package
local dir = os.getenv("DIR")
local archive = dir .. "/tool.zip"

local st = bz.stat(dir .. "/tool")
if st == nil then
  local r = bz.download("https://example.com/tool-" .. os.getenv("GOOS") .. ".zip", archive)
  if not r.ok then error(r.error) end
  r = bz.unzip(archive, dir .. "/tool")
  if not r.ok then error(r.error) end
  bz.chmod(dir .. "/tool/bin/tool", "0755")
end
```

## Pre-run script

Runs on every `bz` invocation while the execution environment is being
resolved (each package's pre-run script runs exactly once per invocation).
It receives the same environment as the install script and can **return a
table** to modify the execution context:

```lua
-- prerun.lua: point the environment at the installed software
local dir = os.getenv("DIR")

return {
  env   = { TOOL_HOME = dir .. "/tool" },        -- extra/overridden env vars
  alias = { tool = "$TOOL_HOME/bin/tool -v" },   -- command aliases
  path  = { dir .. "/tool/bin" },                -- PATH entries (string or list)
}
```

All three keys are optional. `env` entries override resolved variables,
`alias` entries are merged into the alias table, and `path` replaces the
package's PATH contribution. Because it runs before *every* command, keep
pre-run scripts fast and side-effect free — put anything expensive in the
install script.

## Lua API available to scripts

Scripts run in an embedded Lua 5.1 interpreter ([gopher-lua](https://github.com/yuin/gopher-lua))
with the standard library plus:

- `os.getenv(name)` / `os.setenv(name, value)` — read/write the script's
  *isolated* environment (changes are not applied to the real process
  environment); `os.environment` is the full table
- `bz.download(url, destFile)` — HTTP download; returns `{ok, error}`
- `bz.unzip(src, destDir)` / `bz.zip(srcDir, destFile [, {echo=true}])`
- `bz.mkdir(path [, perm])`
- `bz.chmod(path, mode)` — mode as octal string (`"0755"`) or symbolic (`"rwxr-xr-x"`)
- `bz.stat(path)` — returns `{name, size, mode, permission, modtime, isdir}` or `nil`
- `bz.debug(value)` — print a debug value to stdout
