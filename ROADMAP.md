# Roadmap: gtk-ai frente a rtk 0.42.4

Comparación contra [rtk-ai/rtk](https://github.com/rtk-ai/rtk) `0.42.4` (`ba7a9ce`). gtk-ai está en `0.3.3`.

**Decisión de producto (2026-08-17):** gtk sigue el mismo camino que rtk. Reescribe el comando **antes** de ejecutarlo (`PreToolUse` → `git status` se convierte en `gtkai git status`). gtkai ejecuta el binario real, inyecta flags y filtra la salida. El post-filtro solo sobrevive para tools nativas de Claude Code que no pasan por Bash (`Read`, MCP, `Grep`, `Glob`).

---

## 1. Principal — rewrite PreToolUse (proxy)

Hoy gtk filtra *después*. `Module.Rewrite` existe y nunca se llama. Ese es el fallo de diseño, no un desfase de catálogo.

### Flujo objetivo

```text
Sin gtk:     Claude --git status--> shell --> git --> stdout crudo --> Claude

Con gtk:     Claude --git status--> PreToolUse --> gtkai hook-pre
                                                      |
                                                      v
                                         command: gtkai git status
                                                      |
                                                      v
                                         gtkai ejecuta git (flags inyectados)
                                         filtra stdout/stderr
                                         graba gain
                                                      |
                                                      v
                                         Claude recibe salida compacta
```

Claude no ve el rewrite. El agente llama `git status`; el hook lo sustituye por `gtkai git status`.

### Qué implica (y qué no)

| Pieza | Cambio |
|---|---|
| Plugin | Registra `PreToolUse` (matcher `Bash`) además de `PostToolUse`. El plugin sigue siendo quien configura Claude Code. |
| Binario | Pasa a ser proxy CLI: `gtkai git status`, `gtkai ls`, `gtkai hook-pre`. Ejecuta el comando real. |
| `Rewrite()` | Deja de ser código muerto: inyecta flags (`git status --porcelain -b`, `go test -json`, `git log --pretty=…`). |
| `FilterOutput` | Se aplica **dentro** del proxy, sobre la salida que gtkai acaba de capturar. |
| `PostToolUse` | Se queda para `Read`, `mcp__*`, y más adelante `Grep`/`Glob`. No es el camino de Bash. |
| Fases del repo | Siguen dos piezas: plugin registra hooks; binario filtra y ahora también ejecuta. No se mezclan. |

No copiar de rtk: `rtk init` multi-agente, motor TOML, `discover`/`learn`, telemetría, `tee` a disco. gtk sigue siendo heurística (truncado, agrupación, stripping).

### Alcance de esta fase

Un PR (o pocos) que deje el camino de rtk funcionando para **un** comando de punta a punta, y la infraestructura para el resto.

1. **`gtkai hook-pre`**: lee el JSON de PreToolUse, reescribe `tool_input.command` cuando el binario está registrado, escribe `updatedInput`. Si no hay módulo, pasa.
2. **Plugin**: `PreToolUse` + script `gtkai-pre-tool-use.sh`. Matcher `Bash`. Timeout corto.
3. **CLI proxy**: `gtkai <modulo> [args…]` ejecuta el comando, captura stdout+stderr, filtra, imprime, propaga exit code, graba `gain`.
4. **Detección de comando**: no basta la primera palabra. Cubrir `/usr/bin/git`, `sudo git`, `git -C dir status`, `VAR=1 git status`. Pipelines: reescribir solo el último eslabón si es seguro (`grep`/`rg`); si no, pasar.
5. **Guardia `never_worse`**: si el filtrado estima más tokens que el crudo, imprimir el crudo.
6. **Strip ANSI** antes de parsear.
7. **Primer comando extremo a extremo: `git status`**. Rewrite a `gtkai git status`. El módulo inyecta `--porcelain -b` (salvo que el usuario ya pida otro formato) y agrupa por estado.

Criterio de hecho:

- `git status` en Claude Code se reescribe a `gtkai git status` (payload PreToolUse de prueba).
- `gtkai git status` en terminal produce salida compacta y exit code de git.
- `gtkai gain` registra esa invocación.
- Un comando no registrado (`echo hi`) no se toca.

Hasta que esto no esté verde, no se añaden módulos nuevos. El post-filtro actual de Bash se retira cuando el proxy cubra los módulos existentes; mientras tanto puede convivir para no dejar un agujero.

---

## 2. Correcciones — módulos que ya existen y no cumplen

Con el proxy, estos módulos pueden inyectar flags. Eso es lo que hoy no pueden hacer y por eso fallan.

| Módulo | Problema actual | Corrección con rewrite |
|---|---|---|
| `ls` | Reformatea y **sigue listando cada nombre**. El test `TestLsLargeOutputNotModified` documenta que nunca acorta. | Ejecutar `ls` con flags estables (`-la`, `LC_ALL=C`), compactar a recuentos + N ejemplos, omitir dirs de ruido. |
| `git status` | Espera porcelain. Claude lanza formato largo; el parser no entra. | Cubierto en la fase principal. |
| `git diff` | Corte ciego a 300 líneas. | `--stat` + hunks compactos (tope por hunk), strip de `index`/`+++`. |
| `git log` | Corta 50 entradas verbose. | Inyectar `%h %s (%ar) <%an>` si el usuario no pasó `--pretty`/`--oneline`; default 10; respetar `-n`. |
| `git branch` | Parcial. | Filtrar remotes; cap. |
| `grep` | 200 líneas crudas + cabecera. | Mismo contrato que `rg`: grupo por fichero, caps. Parser compartido. |
| `rg` | Casi alineado. | Flags `-A/-B`, pipelines. |
| `find` | 100 paths + extensión. | Agrupar por directorio, cap más bajo, no inflar salidas pequeñas. |
| `Read` | Solo líneas `//` y `#`. | Bloques `/* */`, más extensiones. Sin modo “solo firmas” (agresivo de rtk: peligroso). |
| `gain` | SQLite listo, hook no graba. | Cableado en el runner del proxy (fase 1). |

Añadir al mismo bloque, porque son el mismo camino Bash que rtk reescribe a `read`:

- `git show`, `git add`/`commit`/`push`/`pull`/`fetch`/`stash` (confirmación, sin progress).
- `cat` / `head` / `tail` → `gtkai read` (reutiliza `read.FilterContent`).
- `tree`.

Tools nativas (siguen en `PostToolUse`, rtk no puede):

- `Grep`, `Glob`.
- MCP: mantener truncado + passthrough; compactar JSON cuando el body sea texto.

Criterio de hecho: fixtures de status largo, diff unificado, log verbose, `ls -la`. Ahorro medible en todos, incluido `ls`. `go test ./...` en verde.

---

## 3. Resto — runners y ecosistema

Solo después del proxy y de las correcciones. Aquí el rewrite sí inyecta flags (`go test -json`).

### Runners (mucho ruido, ~90% en rtk)

| Módulo | Comandos | Rewrite | Filtro |
|---|---|---|---|
| `gotest` | `go test`, `go build`, `go vet` | `go test -json` salvo `-bench` o `-json` ya presente | paquetes `ok` → recuento; `FAIL` completo |
| `cargo` | `test`, `build`, `clippy`, `check` | según subcomando | errores/fallos; colapsar `Compiling` |
| `pytest` | `pytest`, `python -m pytest` | — | fallos + traceback corto |
| `npmtest` | `npm test`, `pnpm test`, `npx vitest`/`jest` | — | fallos; strip ANSI |
| `docker` | `ps`, `images`, `logs`, `compose ps/logs` | — | columnas mínimas; logs con tope |

Criterio: fixture `go test` con 40 paquetes ok + 1 FAIL; el agente ve el FAIL y un recuento de los ok.

### Ecosistema, solo si `gain` lo pide

No abrir módulos “por si acaso”:

- linters: `ruff check`, `tsc`, `eslint`/`biome`, `golangci-lint`
- `gh pr`/`issue`/`run`
- `kubectl get`/`logs`
- `curl` JSON (passthrough si el body no es texto)

Fuera de alcance hasta que el núcleo y los runners cubran una sesión típica: filtros TOML de rtk (`helm`, `pulumi`, `terraform`, `dotnet`, `gradle`, `phpunit`, …), multi-agente, `discover`/`learn`.

---

## Estado actual vs objetivo

| | gtk-ai 0.3.3 | Objetivo |
|---|---|---|
| Bash | `PostToolUse` filtra stdout | `PreToolUse` reescribe a `gtkai …`; el binario ejecuta y filtra |
| `Rewrite()` | Muerto | Inyecta flags |
| `Read` / MCP | `PostToolUse` | Se mantiene |
| `gain` | No graba | Cada ejecución del proxy |
| Comandos | find, ls, git (4), grep, rg, Read, MCP | Lo mismo, corregido, luego runners |

El filtrado sigue siendo heurístico. No hay compresión semántica.

---

## Riesgos

- **Contrato de hooks.** Un `PreToolUse` que escriba mal el JSON desactiva el hook (Claude Code lo silencia). Tests con payload real, stdout solo JSON.
- **Pipelines y `&&`.** Reescribir `find … \| head` como `find` rompe el significado. Solo último eslabón seguro, o pasar.
- **Comandos de escritura.** `git commit`, `git push`, `docker run` no pueden quedar a medias. El proxy propaga stdin/TTY cuando el comando no es de solo lectura; si no se puede, no reescribir.
- **Doble filtro.** Mientras convivan Pre y Post sobre Bash, la salida se puede filtrar dos veces y alargar. Retirar el Post de Bash en cuanto el proxy cubra el módulo.
- **`never_worse`.** Un status de 3 ficheros no debe convertirse en un párrafo más largo.
- **Falsos positivos en runners.** Conservar cualquier línea que no se clasifique.

---

## Orden de PRs

1. Proxy PreToolUse + `git status` extremo a extremo (sección 1).
2. Correcciones de módulos actuales + `cat`/`head`/`tail`/`tree` + git restante (sección 2).
3. Runners (sección 3).
4. Ecosistema según `gain`.

No bump de versión hasta que el PR 1 esté usable. El tag tiene que coincidir con todos los ficheros de versión (`cmd/gtkai/main.go`, plugin json, `mcpscan`, README).
