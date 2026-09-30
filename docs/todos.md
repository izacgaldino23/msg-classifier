# TODOs

Backlog of what was deliberately left out of DC-005 (notes, lembretes e listas de tarefas).

## Datas

- [ ] **Datas relativas** — o parser determinístico aceita `dd/mm[/aaaa]`, `dia N`, `hoje`, `amanhã` e `ontem`. Faltam as formas relativas: "daqui a 3 dias", "na sexta", "próxima semana", "semana que vem", "mês que vem".
- [ ] **Weekday parsing** — "sexta", "segunda" → próxima ocorrência daquela semana (precisa de uma noção de "próxima" vs "esta").
- [ ] **Anos ambíguos** — `dd/mm` assume o ano corrente; em janeiro um lembrete de dezembro deveria cair no ano seguinte.
- [ ] **Colisão de substring em `ontem`** — o parser casa `strings.Contains(normalized, "ontem")` na mensagem inteira, e `normalizeName` remove acentos: "contém" vira "contem", que contém "ontem". Uma nota com a frase "o relatório que contém os números" é lida como se fosse de ontem. O casamento precisa ser por palavra inteira (campo a campo), não por substring.
- [ ] **Fuso horário** — as datas são normalizadas para meia-noite UTC. Gravar a data em horário local exigiria um campo separado ou um offset por usuário.

## Lista de tarefas

- [ ] **Deadline na lista** — a lista de tarefas não aceita data (o DC-005 deixa isso para o futuro); quando existir, reusar o `Note.Date` já indexado.
- [ ] **Marcar item como feito** — a coluna `todo_items.done` existe, mas não há caminho de escrita: falta um endpoint/mensagem para alternar o item e o serviço correspondente (`NotesRepository.UpdateItem`).
- [ ] **Título da nota** — nota, lembrete e lista não têm título; separar título de descrição exigiria uma chamada Jev nova (o modelo só suporta choice/noul/score).
- [ ] **Rótulo inicial no primeiro item** — `SplitTodoItems` não remove o rótulo que abre a mensagem: "Tarefas: comprar pão, leite e ovos" persiste "comprar pão" mas o conteúdo original segue a lista, e "Preciso fazer: 1. revisar contrato" deixa "Preciso fazer" grudado no primeiro item quando o texto vem em uma linha só. O comportamento está caracterizado e testado (`TestNoteServiceAddTodoKeepsLeadingLabelOnFirstItem`); corrigir exige detecção de rótulo (`:` no primeiro item) antes do split.

## Busca (require)

- [ ] **Busca insensível a acentos no conteúdo** — hoje `content LIKE %termo%` é insensível a maiúsculas (`LOWER(content)`) mas não a acentos: "acao" não acha "ação". Corrigir com uma coluna `content_norm` (o mesmo padrão de `contacts.name_norm`) ou com uma função SQLite de normalização.
- [ ] **Termo com várias palavras** — a busca remove stopwords e, se a frase não bate, refaz a consulta só com a última palavra. Um índice full-text (FTS5) resolveria o caso geral.
- [ ] **Filtro combinado** — hoje o require escolhe um único filtro (data → pendências → termo). "O que falta para o dia 10?" deveria combinar data + pendência.

## Produto

- [ ] **Confirmação antes de salvar** — no `add` a nota é persistida direto; um "salvar assim?" daria chance de corrigir data/horário.
- [ ] **Exportar/importar notas** — assim como o harness exporta CSV, gerar um `.md`/`.json` das notas e das listas.
- [ ] **Notas por usuário** — o modelo não tem `user_id`; com mais de uma pessoa usando a instância as buscas misturam.