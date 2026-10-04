# Protocolo Operacional de Engenharia Reversa (Bubble to Code)

Este documento atua como **especificação executável e prompt orientador** para qualquer Harness de IA (Antigravity, Claude Code, Cursor, Codex, etc.). Ele descreve o pipeline determinístico para descompilar o arquivo Bubble e gerar a documentação de arquitetura completa em formato `SPEC.md`.

---

## 1. Objetivo da Execução

Transformar um dump bruto de exportação do Bubble (`.bubble`) em:
1. Uma **estrutura fatiada em JSON agnóstica** pronta para consumo de LLMs e agentes.
2. Especificações técnicas formais (**`SPEC.md`**) cobrindo:
   - **Banco de Dados**: Dicionário de dados, mapeamento de tipos e DDL PostgreSQL com FKs e índices.
   - **Workflows**: Gatilhos, condições *Only when*, regras de negócio e services em TypeScript.
   - **UI**: Hierarquia de componentes, formulários, tabelas e código equivalente em React + Tailwind CSS.

---

## 2. Pré-requisitos e Verificação de Arquivos

Certifique-se de que os seguintes arquivos existem no diretório raiz:

```powershell
Get-ChildItem -Path . -Include *.bubble, harness -Recurse
```

- Arquivo `.bubble` (ex: `<seu-app.bubble>`).
- Código-fonte da CLI unificada em `main.go`.
- Submódulos auxiliares em `harness/decompiler/main.go` e `harness/specgen/main.go`.

---

## 3. Passo a Passo de Execução

### Passo 1: Compilar o Executável Nativo

Compile o binário otimizado sem símbolos de depuração (`-ldflags="-s -w"`):

```powershell
go build -ldflags="-s -w" -o bubble-to-code.exe main.go
```

---

### Passo 2: Executar o Pipeline Completo (Decompile + Specs)

Execute o binário unificado passando o arquivo `.bubble` e o diretório de saída:

```powershell
.\bubble-to-code.exe -file <seu-app.bubble> -out extracted_app_harness
```

> **Tempo total esperado:** < 1 segundo (~800ms).  
> **Resultado esperado:** Fatiamento completo em JSON (`01_database`, `02_workflows`, `03_ui_pages`) e geração automática de todos os arquivos `SPEC.md`.

*(Opcional: Se desejar rodar apenas a extração de JSONs, use `.\bubble-to-code.exe -decompile`. Para rodar apenas as specs, use `.\bubble-to-code.exe -specs`).*

---

## 4. Estrutura de Artefatos Gerada e Critérios de Entrega

Ao término da execução, a estrutura entregue deve conter:

```text
extracted_app_harness/
├── app_manifest.json                           <-- Índice consolidado da aplicação
│
├── 01_database/
│   ├── detected_tables.json                    <-- Tabelas customizadas encontradas
│   ├── option_sets.json                        <-- Enums e domínios fixos do Bubble
│   └── SPEC.md                                 <-- Dicionário de dados, mapeamento de tipos e DDL PostgreSQL
│
├── 02_workflows/
│   └── pages/
│       ├── SPEC.md                             <-- Índice consolidado de todos os fluxos
│       ├── <page_name>_workflows.json          <-- Dump bruto de workflows por página
│       ├── <page_name>_SPEC.md                 <-- Especificação direta da página
│       └── <page_name>/
│           └── SPEC.md                         <-- Gatilhos, 'Only when', regras e Services em TypeScript
│
└── 03_ui_pages/
    └── <page_name>/
        ├── elements_tree.json                  <-- Árvore visual bruta da interface
        ├── page_meta.json                      <-- Metadados e Custom States
        └── SPEC.md                             <-- Hierarquia visual, inputs, tabelas e React + Tailwind
```

---

## 5. Validação da Entrega

Execute o comando de verificação para garantir que nenhuma página ficou sem especificação:

```powershell
# Verifica a integridade dos arquivos gerados
Test-Path extracted_app_harness/01_database/SPEC.md
Get-ChildItem -Path extracted_app_harness -Filter SPEC.md -Recurse | Select-Object FullName
```

### Critérios de Aceite:
1. `extracted_app_harness/01_database/SPEC.md` deve conter script SQL DDL funcional com `UUID`, `FOREIGN KEY` e `CREATE INDEX`.
2. Todas as páginas listadas em `app_manifest.json` devem possuir seu respectivo `SPEC.md` em `02_workflows/pages/<página>/` e `03_ui_pages/<página>/`.
3. Os binários Go devem executar em menos de 2 segundos combinados.
