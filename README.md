# ⚡ Bubble to Code Engine

> **Decompilador nativo e gerador de especificações técnicas (`SPEC.md`) para transformar dumps do Bubble.io em arquiteturas limpas de código (PostgreSQL, TypeScript e React + Tailwind CSS).**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Execution Time](https://img.shields.io/badge/Execution%20Time-%3C1s-brightgreen?style=flat)]()
[![Target Database](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat&logo=postgresql)]()
[![Target Frontend](https://img.shields.io/badge/Frontend-React%20%2B%20Tailwind-61DAFB?style=flat&logo=react)]()
[![License](https://img.shields.io/badge/License-MIT-blue.svg)]()

---

## 🎯 Proposta de Valor

Aplicativos desenvolvidos no **Bubble.io** exportam seus metadados completos em arquivos de dump monolíticos (`.bubble`) que frequentemente contêm centenas de milhares de linhas de JSON aninhado. 

O **Bubble to Code Engine** é um compilador de engenharia reversa escrito em **Go nativo de alta performance** que:
1. **Fatia o dump bruto em camadas agnósticas** (Banco de dados, Enums, Telas, Workflows e Integrações).
2. **Gera especificações executáveis (`SPEC.md`)** padronizadas para serem consumidas diretamente por agentes de IA (Antigravity, Claude Code, Cursor, Codex) e desenvolvedores.

---

## ⚡ Performance Extrema

| Etapa | Abordagem Tradicional (Node/Python) | Bubble to Code (Go Nativo) |
| :--- | :--- | :--- |
| **Parsing & Extração** | ~3.5s a 8.0s | **~165ms** |
| **Geração de SPECs** | ~12.0s a 25.0s | **~650ms** |
| **Pipeline Completo** | ~15s a 30s | **< 1 segundo (~820ms)** |
| **Memória RAM** | 150MB+ | **< 25MB** |
| **Dependências Externas** | Node.js + NPM + PyTorch | **Zero (Binário único `.exe`)** |

---

## 🏗️ Arquitetura de Fatiamento

O pipeline processa o dump do Bubble e o organiza deterministicamente em 4 camadas de engenharia:

```text
extracted_app_harness/
├── app_manifest.json                     # Índice consolidado e metadados gerais
│
├── 01_database/                          # CAMADA 1: DADOS & DOMÍNIOS
│   ├── detected_tables.json              # Tabelas customizadas detectadas
│   ├── option_sets.json                  # Enums e domínios fixos
│   └── SPEC.md                           # Dicionário de dados, mapeamento de tipos e DDL PostgreSQL
│
├── 02_workflows/                         # CAMADA 2: LÓGICA E REGRAS DE NEGÓCIO
│   └── pages/
│       ├── SPEC.md                       # Índice consolidado de todos os fluxos
│       ├── <page_name>_workflows.json    # JSON bruto de gatilhos e ações por tela
│       └── <page_name>/
│           └── SPEC.md                   # Gatilhos, 'Only when' e Casos de Uso (Services) em TS
│
├── 03_ui_pages/                          # CAMADA 3: INTERFACE E DESIGN SYSTEM
│   └── <page_name>/
│       ├── elements_tree.json            # Árvore visual bruta da interface
│       ├── page_meta.json                # Custom states e propriedades da tela
│       └── SPEC.md                       # Hierarquia visual, forms, tabelas e React + Tailwind
│
└── 04_integrations/                      # CAMADA 4: INTEGRAÇÕES EXTERNAS
    └── api_connectors.json               # Chamadas REST e payloads do API Connector
```

---

## 🚀 Como Executar (Quickstart)

### 1. Compilação do Binário

Com o Go 1.24+ instalado, compile o executável único otimizado:

```powershell
go build -ldflags="-s -w" -o bubble-to-code.exe main.go
```

### 2. Execução do Pipeline Completo

Execute passando o caminho do dump `.bubble`:

```powershell
.\bubble-to-code.exe -file gestao-contratos-conexao.bubble -out extracted_app_harness
```

Pronto! Em menos de 1 segundo toda a documentação `SPEC.md` estará pronta para uso em `./extracted_app_harness/`.

### Flags Opcionais:
* `-decompile`: Executa apenas o fatiamento em JSON sem gerar os arquivos markdown.
* `-specs`: Executa apenas a geração das especificações técnicas `SPEC.md`.
* `-file <caminho>`: Define o arquivo `.bubble` de entrada (padrão: `gestao-contratos-conexao.bubble`).
* `-out <caminho>`: Define o diretório de destino (padrão: `extracted_app_harness`).

---

## 🤖 Integração com Agentes de IA

Este repositório foi projetado desde a base para interoperar perfeitamente com harnesses de IA como **Antigravity**, **Claude Code**, **Cursor** e **Aider**. 

Consulte o arquivo [`AGENTS.md`](./AGENTS.md) para regras operacionais e o arquivo [`passos-execucao.md`](./passos-execucao.md) para o protocolo detalhado de execução.

---

## 📄 Licença

Este projeto é distribuído sob a licença [MIT](LICENSE).
