-- Wipe and re-seed the Jev validation examples.
DELETE FROM jev_prompts;

INSERT INTO jev_prompts (flow, message, expected_result) VALUES
('classification', 'Salva o telefone do João: (11) 91234-5678', 'contact:add'),
('classification', 'Adiciona o email da Maria: maria@exemplo.com', 'contact:add'),
('classification', 'Quero o contato do Pedro, me passa o número dele', 'contact:require'),
('classification', 'Me lembra de pagar a conta de luz amanhã', 'notes:add'),
('classification', 'Quanto gastei com mercado esse mês?', 'finance:require'),
('classification', 'Agenda reunião com o time às 14h', 'schedule:add'),
('classification', 'Que horas é minha consulta na sexta?', 'schedule:require'),
('classification', 'Anota que preciso comprar pão', 'notes:add'),
('classification', 'Me mostra minhas anotações sobre o projeto', 'notes:require'),
('classification', 'Isso não é nada importante', 'other:add'),
('name', 'Salva o contato do João da Silva, telefone (11) 91234-5678', 'João da Silva'),
('name', 'Adiciona Maria Clara Oliveira ao catálogo', 'Maria Clara Oliveira'),
('name', 'Cadastra o Pedro Henrique Santos', 'Pedro Henrique Santos'),
('name', 'Guarda o número da Ana Beatriz', 'Ana Beatriz'),
('name', 'Contato do José Carlos de Souza', 'José Carlos de Souza'),
('name', 'Salva a Ana Paula, email ana@exemplo.com', 'Ana Paula');