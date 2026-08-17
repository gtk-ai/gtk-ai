# Roadmap: gtk-ai frente a rtk 0.42.4

Propuesta de actualización. No implementa código: describe el desfase, prioriza por ahorro de tokens y pide validación antes de aplicar cambios.

Comparación hecha contra [rtk-ai/rtk](https://github.com/rtk-ai/rtk) `0.42.4` (`ba7a9ce`, rama `develop`, 2026-06). gtk-ai está en `0.3.3`.

## Conclusión

gtk no está “unas versiones atrás”. Es un subconjunto de rtk con otra arquitectura. rtk intercepta el comando **antes** de ejecutarlo (`PreToolUse` → `git status` se convierte en `rtk git status`) e inyecta flags (`--porcelain`, `go test -json`). gtk filtra el stdout **después** (`PostToolUse`).

Eso limita lo que se puede copiar tal cual. También da a gtk ventajas que rtk no tiene: filtra la herramienta nativa `Read`, trunca MCP y puede cubrir `Grep`/`Glob` de Claude Code. rtk lo dice en su README: esas tools no pasan por su hook.

El mayor ahorro no está en portar los 100+ comandos de rtk. Está en:

1. hacer que los módulos actuales reduzcan de verdad (hoy `ls` no ahorra nada)
2. compactar git/grep/read al nivel de rtk
3. añadir runners de test/build (`go test`, `pytest`, `cargo test`, `npm test`) — rtk declara 90%+ ahí
4. decidir si gtk sigue siendo solo post-filtro o también reescribe comandos

---

## Arquitectura: qué no hay que copiar a ciegas

| | gtk-ai 0.3.3 | rtk 0.42.4 |
|---|---|---|
| Modelo | Post-filtro heurístico | Proxy CLI: ejecuta el binario real y comprime |
| Hook | `PostToolUse` (`Bash\|mcp__.*\|Read`) | `PreToolUse` rewrite transparente |
| `Rewrite()` | Existe en la interfaz; **nunca se llama** | Núcleo del producto |
| Alcance | Claude Code | Claude, Cursor, Copilot, Gemini, Codex, Windsurf, … |
| Comandos | find, ls, git (4 subcmds), grep, rg, Read, MCP | 100+ + ~60 filtros TOML |
| Métrica | `bytes/4`, `gain` **no está cableado al hook** | SQLite en cada ejecución |
| Guardia | Algunos módulos comparan `len(filtered) >= len(raw)` | `never_worse` global por tokens estimados |
| stderr | Se ignora | Se fusiona o se deja pasar según el módulo |
| Pipelines | Solo la primera palabra del comando | Reescribe el último eslabón si es seguro |

gtk debe seguir siendo filtrado heurístico (truncado, agrupación, stripping). No hay que importar “compresión semántica” ni el motor TOML de rtk hasta que el núcleo esté sólido.

Copiar el proxy `PreToolUse` cambiaría el contrato del proyecto (el binario pasaría a ejecutar comandos). Es una decisión de producto, no un port mecánico.

---

## Qué hace ya gtk (y dónde falla)

### Cubierto, con huecos

| Módulo | Qué hace gtk | Qué hace rtk | Hueco |
|---|---|---|---|
| `find` | Corta a 100 líneas, agrupa por extensión | Agrupa por directorio, tope 50, ignora ruido (`node_modules`, …) | gtk muestra casi todos los paths; el ahorro real es bajo |
| `ls` | Agrupa por extensión **y sigue listando cada nombre** | Árbol compacto, tamaños, octal, omite dirs de ruido | El test `TestLsLargeOutputNotModified` documenta que **el reformateo nunca es más corto**. El módulo no ahorra tokens |
| `grep` | Corta a 200 líneas + cabecera de recuento | Agrupa por fichero, tope por fichero, tee del resto | gtk deja las 200 líneas enteras; rtk colapsa |
| `rg` | Agrupa por fichero, 10 matches/fichero, 20 ficheros | Mismo enfoque, más fiel a flags y contexto `-A/-B` | gtk está más cerca; faltan contexto y pipelines |
| `git diff` | Corta a 300 líneas crudas | `--stat` + hunks compactos (100 líneas/hunk), strip de cabeceras `index`/`+++` | gtk no compacta: solo trunca el final |
| `git log` | Corta a 50 entradas del formato verbose | Inyecta `%h %s (%ar) <%an>`, default 10, sin merges | Sin rewrite, gtk puede compactar el texto verbose a una línea/commit |
| `git status` | Espera porcelain (` M`, `??`) | Inyecta `--porcelain -b` y agrupa por estado | Claude suele lanzar `git status` largo. El parser de gtk **no lo entiende** y a menudo no filtra |
| `git branch` | Lista local + current | Filtra remotes, caps | Parcialmente alineado |
| `Read` | Quita líneas `//` y `#`, colapsa blancos, corta a 300 | Niveles none/minimal/aggressive; bloques `/* */`; firmas en aggressive | gtk no quita bloques ni docstrings; no hay modo firmas |
| MCP | Trunca a 3000 chars + passthrough por env | No aplica (no es PostToolUse) | Ventaja de gtk |

### Declarado y muerto

- `Module.Rewrite`: todos los módulos devuelven `(nil, false)`. No hay `PreToolUse`.
- `gtkai gain`: SQLite listo; el hook **no graba** `tokens_in`/`tokens_out`. El dashboard siempre está vacío en uso real.

### Tools de Claude Code que gtk no toca

rtk no puede. gtk sí podría, porque ya vive en `PostToolUse`:

- `Grep` (herramienta nativa, no `Bash("grep …")`)
- `Glob`
- `WebFetch` / `WebSearch` (truncado defensivo, como MCP)

---

## Inventario rtk: qué merece la pena

rtk agrupa comandos en ecosistemas. Para gtk, el criterio es **frecuencia en sesiones Claude Code × ruido de stdout**, no paridad de catálogo.

### Alta prioridad (mucho ruido, poco parser)

Aplicable como post-filtro, sin ejecutar el comando:

| Comando | Estrategia rtk | Ahorro declarado | En gtk |
|---|---|---|---|
| `git status/diff/log/show/branch` | compact + caps | 70–80% | parcial, débil |
| `git add/commit/push/pull/fetch/stash` | una línea de confirmación, strip progress | 59–92% | ausente |
| `ls` / `tree` | recuentos + árbol, sin listar todo | 65–80% | `ls` inútil; `tree` ausente |
| `cat` / `head` / `tail` | mismo pipeline que `read` | ~70% | ausente (solo tool `Read`) |
| `grep` / `rg` | agrupar + cap por fichero | ~75% | grep débil; rg ok |
| `find` | agrupar por dir, cap 50 | ~70% | truncado plano |
| `go test` / `go build` / `go vet` | fallos only; rtk inyecta `-json` | 75–90% | ausente |
| `cargo test/build/clippy` | fallos / errores, colapsa ok | 80–90% | ausente |
| `pytest` | fallos + traceback recortado | ~90% | ausente |
| `npm test` / `pnpm test` / `vitest` / `jest` | fallos only | 90–99% | ausente |
| `docker ps/images/logs` | columnas esenciales | ~85% | ausente |
| `gh pr/issue/run` | vista compacta | 80–87% | ausente |

`go test` es el caso límite: rtk inyecta `-json` en PreToolUse. En PostToolUse se puede parsear la salida texto (`FAIL`, `--- FAIL`, `ok`/`PASS`) y colapsar paquetes verdes. Menos preciso, suficiente, y no exige proxy.

### Media prioridad (útil si el usuario trabaja en ese stack)

`ruff`, `tsc`, `eslint`/`biome`, `golangci-lint`, `kubectl get/logs`, `curl` (JSON compacto, no binarios), `playwright`, `next build`, `prisma`, `make`.

### Baja prioridad / no copiar ahora

Filtros TOML de rtk (`helm`, `pulumi`, `terraform`, `gcloud`, `phpunit`, `dotnet`, `gradle`, `ansible`, `shopify`, `fail2ban`, …): son decenas de parsers de un solo comando. gtk no tiene DSL; cada uno es un módulo Go. No compensan hasta que el núcleo y los runners cubran el 80% de una sesión típica.

Tampoco: multi-agente (`rtk init --agent cursor|gemini|…`), `discover`/`learn` sobre historial de Claude, permisos de rewrite, telemetría, `tee` a disco para recuperar output truncado. Son producto rtk, no token-killers del hook actual.

---

## Fases propuestas

Cada fase es un PR (o pocos PRs) independiente. No subir versión hasta que una fase esté usable. No mezclar proxy PreToolUse con filtros nuevos en el mismo cambio.

### Fase 0 — Cimientos (sin esto el resto miente)

Objetivo: que cada filtro sea seguro, medible y reciba el comando real.

1. **Guardia global `never_worse`**: si el filtrado estima más tokens que el crudo, devolver el crudo. Hoy `find` y `grep` pueden alargar salidas pequeñas; `ls` siempre alarga.
2. **Pasar args al módulo**: `FilterOutput(output string)` no basta. git ya hace un caso especial. El registry debería exponer `FilterOutput(args []string, output string)` para que `git log --oneline` no se trate igual que `git log`.
3. **stderr**: el hook solo lee `stdout`. `go test`, `cargo`, `git` y linters tiran ruido (o fallos) por stderr. Concatenar o filtrar ambos.
4. **Strip ANSI** antes de parsear.
5. **Cablear `gain` en el hook**: un `Record` por invocación filtrada. Sin esto no hay forma de saber si las fases siguientes sirven.
6. **Detección de comando**: primera palabra es frágil (`cd foo && git status`, `sudo git`, `/usr/bin/git`, `git -C dir status`). Extraer el binario real y el subcomando git, como hace rtk con opciones globales.

Criterio de hecho: `gtkai gain` muestra datos tras una sesión; ningún filtro empeora el recuento estimado.

### Fase 1 — Paridad de los módulos que ya existen

Máximo ahorro por línea de código: gtk ya intercepta estos comandos.

1. **`ls`**: dejar de listar todos los nombres. Recuento por extensión + dirs; mostrar N ejemplos como mucho. Si no se puede acortar, no reescribir (el test actual lo deja por escrito).
2. **`git status`**: parsear formato largo *y* porcelain. Agrupar staged / modified / untracked / deleted. Quitar hints `(use "git …")`.
3. **`git diff` / `git show`**: compactar hunks (cabecera de fichero + `+/-` + tope por hunk), no un corte ciego a 300 líneas.
4. **`git log`**: compactar cada commit a `hash subject (fecha) <autor>`; respetar `--oneline`/`--format` del usuario (no empeorar lo que ya es corto).
5. **`git add/commit/push/pull/fetch`**: una línea de resultado; tirar progress `Enumerating objects`.
6. **`grep`**: mismo contrato que `rg` (grupo por fichero, caps). Compartir parser.
7. **`find`**: agrupar por directorio; cap más agresivo; no inflar salidas pequeñas.
8. **`Read` + `cat`/`head`/`tail` por Bash**: bloques `/* */`, más extensiones, colapso de blancos. `cat`/`head`/`tail` reutilizan `read.FilterContent`.
9. **Tools nativas `Grep` y `Glob`**: extender el matcher del plugin. rtk no puede; gtk sí.

Criterio de hecho: repetir `go test ./internal/hook/...` con fixtures reales (status largo, diff unificado, log verbose, `ls -la`). Ahorro medible en todos, incluido `ls`.

### Fase 2 — Runners (el 90% que gtk no toca)

Un módulo por familia, todos post-filtro, fallos only:

| Módulo | Comandos | Regla |
|---|---|---|
| `gotest` | `go test`, `go build`, `go vet` | paquetes `ok` → recuento; dejar `FAIL` + output del test |
| `cargo` | `cargo test`, `build`, `clippy`, `check` | errores/fallos; colapsar `Compiling`/`Finished` |
| `pytest` | `pytest`, `python -m pytest` | fallos + traceback corto; `passed` → recuento |
| `npmtest` | `npm test`/`run test`, `pnpm test`, `npx vitest`/`jest` | fallos; strip ANSI; colapsar pass |
| `docker` | `docker ps`, `images`, `logs`, `compose ps/logs` | columnas mínimas; logs con tope |

Opcional en la misma fase: `tree` (cap de profundidad + ignore de ruido).

No inyectar `-json` todavía. Parsear texto. Si más adelante hay PreToolUse, se mejora `go test` sin cambiar el filtro.

Criterio de hecho: fixture de `go test ./...` con 40 paquetes ok + 1 FAIL; el agente ve el FAIL completo y un recuento de los ok.

### Fase 3 — Decisión de producto: ¿PreToolUse?

Hasta aquí gtk sigue siendo post-filtro. Para igualar rtk en `go test -json`, `git status --porcelain` y `cat`→filtro de read **antes** de generar output enorme, hace falta rewrite.

Opciones:

- **A (recomendada a corto plazo)**: no. Seguir en PostToolUse. El ahorro de las fases 0–2 cubre la mayoría de sesiones sin ejecutar comandos desde gtkai.
- **B**: hook `PreToolUse` que reescribe `git status` → `git status --porcelain -b`, `go test` → `go test -json`, etc., **sin** que gtkai ejecute el binario. El plugin reescribe args; el filtro post sigue igual. Encaja con “el binario no configura Claude Code”: el rewrite vive en el plugin o en `gtkai hook-pre` que solo imprime JSON `updatedInput`.
- **C**: proxy estilo rtk (`gtkai git status` ejecuta git). Rompe el modelo actual. No hacerlo salvo decisión explícita.

Si se elige B: implementar `Rewrite` de verdad, registrar `PreToolUse` en `plugin/hooks/hooks.json`, y no mezclar con filtros nuevos.

### Fase 4 — Ecosistema, solo si el `gain` lo pide

Cuando `gtkai gain` muestre qué comandos aparecen de verdad:

- linters: `ruff check`, `tsc`, `eslint`/`biome`, `golangci-lint` (agrupar por regla/fichero)
- `gh pr/issue/run`
- `kubectl get/logs`
- `curl` JSON (estructura + tope; **passthrough si el body no es texto**)
- MCP: truncado por tipo (JSON compacto vs corte a 3000 chars)

No abrir un módulo “por si acaso”. Cada uno exige parser y tests de fixture.

---

## Qué gtk ya hace mejor que rtk (mantener)

- Filtrado de `Read` nativo de Claude Code.
- Truncado MCP + `GTK_MCP_PASSTHROUGH_PATTERNS` + `gtkai mcp-scan`.
- Plugin de dos piezas (binario ≠ config de Claude), más simple de auditar.
- Superficie pequeña: se puede endurecer el núcleo antes de inflar el catálogo.

No diluir esto con `rtk init` multi-agente ni con 60 TOML.

---

## Riesgos

- **Post-filtro no puede inyectar flags.** `go test -json` y `git status --porcelain` no ocurrirán solos. Los parsers tienen que aceptar la salida que Claude pide.
- **Pipelines y `&&`.** Filtrar `find … | head` como `find` puede romper el significado. Fase 0: o se detecta pipe y se pasa, o se filtra solo el último comando seguro (`grep`/`rg`).
- **Falsos positivos en test runners.** Colapsar un `PASS` que esconde un panic. Conservar cualquier línea que no se clasifique.
- **`never_worse` vs reformateo.** Un status de 3 ficheros no debe convertirse en un párrafo más largo.
- **Read agresivo.** Quitar cuerpos de función (modo aggressive de rtk) es heurística peligrosa. En gtk, dejarlo fuera de Fase 1.

---

## Orden de implementación sugerido

Si se valida este roadmap, el primer código sería Fase 0 + `ls`/`git status`/`git diff` de Fase 1. Eso arregla módulos que hoy no cumplen su README, antes de añadir `go test`.

No bump de versión hasta cerrar Fase 0. El tag tiene que coincidir con todos los ficheros de versión (`cmd/gtkai/main.go`, plugin json, `mcpscan`, README).
