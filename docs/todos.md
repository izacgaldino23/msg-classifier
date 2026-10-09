# TODOs

Backlog of what was deliberately left out of DC-005 (notes, lembretes e listas de tarefas) e do DC-007 (finanças).

## Datas

- [x] **Datas relativas** — resolvido no DC-007: o `dateparse.go` agora entende `semana passada`/`última semana`/`semana anterior`, `semana que vem`/`próxima semana`, `mês passado`/`mês anterior`/`último mês`, `mês que vem`/`próximo mês`, `esse mês`/`este mês` e o par `10 de outubro de 2023`/`15 de novembro`. Um `ParseRange` devolve a janela inteira (`from`/`until`), que é o que o require das finanças usa para "esse mês".
- [ ] **Weekday parsing** — "sexta", "segunda" → próxima ocorrência daquela semana (precisa de uma noção de "próxima" vs "esta"). As fases acima param em semana e mês, não em dia da semana.
- [ ] **Contagem a partir de hoje** — "daqui a 3 dias" e "em duas semanas" ainda não são entendidos; só as fases prontas ("próxima semana", "mês que vem") são.
- [ ] **Anos ambíguos** — `dd/mm` assume o ano corrente; em janeiro um lembrete de dezembro deveria cair no ano seguinte.
- [ ] **Colisão de substring em `ontem`** — o parser casa `strings.Contains(normalized, "ontem")` na mensagem inteira, e `normalizeName` remove acentos: "contém" vira "contem", que contém "ontem". Uma nota com a frase "o relatório que contém os números" é lida como se fosse de ontem. O casamento precisa ser por palavra inteira (campo a campo), não por substring.
- [ ] **Fuso horário** — as datas são normalizadas para meia-noite UTC. Gravar a data em horário local exigiria um campo separado ou um offset por usuário.

## Finanças

- [ ] **Extração de imagem/PDF** — hoje o classificador só recebe texto (`ReceiveMessageRequest.Message`). O próximo passo é o usuário poder enviar um comprovante (foto ou PDF) e o bot extrair valor, data e estabelecimento. Antes de codar: upload + OCR, ou mandar o arquivo direto para um modelo multimodal? A parte determinística (`ParseAmount`, `ParseEventDate`, regex do party) continua valendo nos dois caminhos.
- [ ] **Party sem preposição** — o candidato do estabelecimento nasce de uma regex de preposição, então "padaria, 30 reais" não gera party nenhum (a transação é salva, mas sem estabelecimento). Pedir ao Jev o party como substring entre aspas resolve.
- [ ] **Parcelamento** — "3 vezes de 300 reais" guarda 300 e o resto fica no texto original, por decisão do DC-007. Quando parcelamento virar consulta de verdade, precisa de uma coluna própria (`installments`), e não de um amount negativo.
- [ ] **Party no harness** — o flow `finance` do `/prompts` compara **só o tipo**, porque o party é a parte mais fuzzier e sujaria o CSV. Quando o party estabilizar, ele entra como coluna do CSV de evaluation.

## Lista de tarefas

- [ ] **Deadline na lista** — a lista de tarefas não aceita data (o DC-005 deixa isso para o futuro); quando existir, reusar o `Note.Date` já indexado.
- [ ] **Marcar item como feito** — a coluna `todo_items.done` existe, mas não há caminho de escrita: falta um endpoint/mensagem para alternar o item e o serviço correspondente (`NotesRepository.UpdateItem`).
- [ ] **Título da nota** — nota, lembrete e lista não têm título; separar título de descrição exigiria uma chamada Jev nova (o modelo só suporta choice/noul/score).
- [x] **Rótulo inicial no primeiro item** — resolvido no IMP-004: `SplitTodoItems` descarta o rótulo que abre a mensagem (`Tarefas:`, `Comprar:`) quando há lista depois dele, e o " e " passa a separar itens quando a lista já se declarou lista por vírgula/ponto-e-vírgula ou por rótulo. Listas por linha ou numeração seguem com um item por linha.

## Busca (require)

- [ ] **Busca insensível a acentos no conteúdo** — hoje `content LIKE %termo%` é insensível a maiúsculas (`LOWER(content)`) mas não a acentos: "acao" não acha "ação". Corrigir com uma coluna `content_norm` (o mesmo padrão de `contacts.name_norm`) ou com uma função SQLite de normalização.
- [ ] **Termo com várias palavras** — a busca remove stopwords e, se a frase não bate, refaz a consulta só com a última palavra. Um índice full-text (FTS5) resolveria o caso geral.
- [ ] **Filtro combinado** — hoje o require escolhe um único filtro (data → pendências → termo). "O que falta para o dia 10?" deveria combinar data + pendência.
- [ ] **Telefone com separadores na busca** — a busca de contato (require e `/data`) casa o telefone como ele está gravado, só dígitos: "11 98888-7777" não acha "11988887777". Ou o termo que parece telefone perde os separadores antes do `LIKE`, ou uma coluna `phone_digits` indexada.

## Produto

- [ ] **Confirmação antes de salvar** — no `add` a nota é persistida direto; um "salvar assim?" daria chance de corrigir data/horário.
- [ ] **Exportar/importar notas** — assim como o harness exporta CSV, gerar um `.md`/`.json` das notas e das listas.
- [ ] **Notas por usuário** — o modelo não tem `user_id`; com mais de uma pessoa usando a instância as buscas misturam.