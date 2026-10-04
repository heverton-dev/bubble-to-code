# 🤖 AGENTS.MD: Protocolo Operacional para Agentes de IA

Este documento define as **regras inegociáveis, padrões de comportamento e instruções técnicas** para qualquer Agente de IA (Antigravity, Claude Code, Cursor, Codex, Aider) que opere neste repositório.

---

## 1. Regras Fundamentais e Inegociáveis

1. **Idioma Oficial**: Toda comunicação técnica, geração de planos, resumos e artefatos de documentação devem ser gerados em **Português do Brasil (PT-BR)**.
2. **Eficiência e Economia de Tokens**:
   - Vá direto ao ponto. Respostas concisas e estruturadas.
   - Use tabelas e listas em vez de parágrafos extensos.
   - Não repita o que acabou de ser feito se o diff já demonstra a alteração.
3. **Padrão Tecnológico do Núcleo**:
   - O núcleo do motor é **Go 1.24+ nativo**.
   - **NÃO** crie scripts temporários em Node.js ou Python quando uma solução em Go puder ser compilada e executada diretamente.
   - O binário deve ser sempre compilado com `-ldflags="-s -w"`.
4. **Proteção de Dados & Git**:
   - Arquivos `.bubble`, diretórios `extracted_app*` e binários `.exe` **NUNCA** devem ser adicionados ao rastreamento do Git.
   - Verifique sempre o status do [`.gitignore`](./.gitignore) antes de realizar commits.

---

## 2. Comandos Canônicos de Execução

Ao interagir com o terminal neste repositório, utilize exclusivamente os comandos canônicos:

```powershell
# 1. Compilação do binário unificado
go build -ldflags="-s -w" -o bubble-to-code.exe main.go

# 2. Execução do pipeline completo (Decompile + SPEC.md)
.\bubble-to-code.exe -file <seu-app.bubble> -out extracted_app_harness

# 3. Execução modular (se necessário)
.\bubble-to-code.exe -decompile -file <seu-app.bubble> -out extracted_app_harness
.\bubble-to-code.exe -specs -file <seu-app.bubble> -out extracted_app_harness
```

---

## 3. Como Consumir os Artefatos Gerados para Implementar Código

Quando for solicitado a codificar ou migrar uma funcionalidade a partir do dump do Bubble:

1. **Para implementar o Banco de Dados**:
   - Leia [`extracted_app_harness/01_database/SPEC.md`](./extracted_app_harness/01_database/SPEC.md).
   - Utilize o script DDL PostgreSQL já gerado com chaves estrangeiras e índices.
2. **Para implementar Regras de Negócio e Backend**:
   - Acesse [`extracted_app_harness/02_workflows/pages/<nome_da_tela>/SPEC.md`](./extracted_app_harness/02_workflows/pages/).
   - Analise os gatilhos e as condições *Only when* antes de escrever os Services ou Endpoints.
3. **Para implementar Telas e Componentes Frontend**:
   - Acesse [`extracted_app_harness/03_ui_pages/<nome_da_tela>/SPEC.md`](./extracted_app_harness/03_ui_pages/).
   - Utilize a proposta em React + Tailwind CSS como fundação para a tela.

---

## 4. Checklist de Validação do Agente

Antes de finalizar qualquer tarefa neste repositório, valide os seguintes pontos:
- [ ] O código Go compila sem warnings (`go build ./...`).
- [ ] O `git status` está limpo e nenhum arquivo listado no `.gitignore` foi indexado acidentalmente.
- [ ] As especificações `SPEC.md` refletem fielmente as estruturas do arquivo `.bubble`.
- [ ] O tempo de execução da ferramenta permaneceu abaixo de 1 segundo.
