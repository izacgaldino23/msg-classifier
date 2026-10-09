# Message Classifier

## Project info

To learn fast, check ARCHITECTURE.md and CODE_STYLE.md.
- `/prompts` — Jev validation harness (examples in `scripts/sql/seed_prompts.sql`, CSV exports to `exports/`).
- `/data` — data screen (DC-006): browse/edit/delete contacts, notes and transactions, no Jev calls.
- `docs/diagrams/` — os diagramas de fluxo em PlantUML (fonte versionado; nenhuma imagem).

## After editing the code

After any code change, remember to update the above md files and this one, if necessary. Commit them as well.

Also after implementing anything from docs/decisions, add a subheading with a summary with the implemented code, only if the DC file doesnt have specifing it.

## Keep on track

- Dont add too verbose comments or unnecessary ones. 
- Keep in mind to use MVC standard.
- Never execute the plan before asking confirmation from user.

## Text

- Todo texto que o usuario le vem de `internal/messages` (locales/*.json). Nunca hardcodar portugues em Go, e nunca chamar uma chave crua no call site: use a funcao tipada (`messages.ContactSaved(id)`). Layout (indentacao, HTML, `": "`) continua na superficie.

## Diagramas de fluxo

O fluxo vive em `docs/diagrams/*.puml` (PlantUML, so o fonte — nenhuma imagem e gerada ou commitada). Se voce mudar o fluxo — uma categoria, uma action, uma rota, um comando, um filtro, uma tela — **atualize o diagrama na mesma mudanca**, senao a visao geral passa a mentir.

- `visao-geral.puml` — os tres entrypoints (web · api · cli) sobre um core so.
- `fluxo-mensagem.puml` — o limite de uma mensagem: classificacao, contact · notes · finance (add x require) e o loop de duplicado.
- `fluxo-dados.puml` — a tela `/data` (e quem mais le o mesmo DataService).
- `fluxo-cli.puml` — o REPL: comando x mensagem.

Valide a sintaxe com `java -jar plantuml.jar -checkonly docs/diagrams/*.puml`.

## Git

- Never use git worktrees — I'm the only one working in this repo. Work directly on a branch in the main checkout and push from there.

## DC (decisions) execution

After planning and when the usar start executing the plan, first add a subheading with more, human like, information about how this task it'll be executed. Remember to keep the edits in those files simples. You dont need to refer all files, I think that a small but complete explanation must work.

### Gaps

If the user DC has some gap after brainstorm, dont be shy and ask the user how the best way of fill each gap before planning. (use question tool for this).

## TODO list

When planning remember to create a TODO list to be following through the execution, must be updated after each step is ready.