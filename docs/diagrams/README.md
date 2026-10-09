# Diagramas de fluxo

O fluxo do sistema em PlantUML. **Só o fonte `.puml` é versionado — nenhuma imagem é gerada ou commitada.**

| Arquivo | O que mostra |
|---|---|
| `visao-geral.puml` | os três entrypoints (web · api · cli) sobre um core só |
| `fluxo-mensagem.puml` | o limite de uma mensagem: classificação, as três categorias, add × require, o loop de duplicado |
| `fluxo-dados.puml` | a tela `/data`: filtro, busca, ver, editar, apagar |
| `fluxo-cli.puml` | o REPL: cada linha vira um comando ou uma mensagem |

Como ver: extensão **PlantUML** no VS Code (preview com `Alt+D`), ou cole o fonte em <https://www.plantuml.com/plantuml>. Para validar a sintaxe sem abrir nada:

```bash
java -jar plantuml.jar -checkonly docs/diagrams/*.puml
```

Ao mudar o fluxo, atualize o diagrama correspondente na mesma mudança — é a regra que está no `CLAUDE.md`.
