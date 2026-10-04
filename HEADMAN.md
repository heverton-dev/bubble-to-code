# 👑 HEADMAN: Governança Executiva & Arquitetura do Projeto

**Projeto:** `bubble-to-code`  
**Responsável Técnico / Headman:** Lead Software Engineer & AI Harness Orchestrator  
**Status do Projeto:** Production Ready (Versão 1.0.0 Nativa Go)  

---

## 1. Visão Estratégica e Liderança do Projeto

O propósito do **`bubble-to-code`** é libertar produtos consolidados no Bubble.io de lock-in tecnológico, fornecendo uma transição transparente e determinística para arquiteturas modernas de código aberto (PostgreSQL, NestJS/Node/Go e Next.js/React).

Como Headman deste repositório, os pilares inegociáveis de engenharia são:
1. **Velocidade Bruta (Raw Performance)**: O pipeline completo nunca deve ultrapassar 2 segundos de execução total em máquinas padrão.
2. **Determinismo Estrito**: Dumps idênticos devem gerar especificações e schemas idênticos, garantindo rastreabilidade absoluta para agentes de IA.
3. **Zero Dependências Pesadas**: A distribuição principal deve sempre ser um binário compilado único em Go (`.exe` no Windows, binário ELF no Linux).
4. **Isolamento de Dados Sensíveis**: Nomes de clientes, tokens e arquivos brutos `.bubble` nunca devem ser comitados no controle de versão (sempre protegidos via `.gitignore`).

---

## 2. Mapa de Responsabilidades e Camadas Técnicas

| Camada | Responsabilidade Técnica | Saída Esperada |
| :--- | :--- | :--- |
| **Camada 1: Database & Schemas** | Mapeamento de tipos Bubble -> PostgreSQL, normalização relacional, PKs UUID, FKs com constraints e índices. | `01_database/SPEC.md` |
| **Camada 2: Business Logic & Workflows** | Mapeamento de gatilhos visuais, condições guardas (*Only when*) e propostas de Services tipados com Zod em TypeScript. | `02_workflows/pages/<page>/SPEC.md` |
| **Camada 3: UI & Design System** | Desconstrução de elementos visuais, mapeamento de formulários, inputs e tabelas equivalentes em React + Tailwind CSS. | `03_ui_pages/<page>/SPEC.md` |
| **Camada 4: External Integrations** | Extração de chamadas HTTP, endpoints, headers e payloads do Bubble API Connector. | `04_integrations/api_connectors.json` |

---

## 3. Diretrizes para Contribuições e Evoluções Futuras

1. Qualquer novo parser ou otimização deve ser implementado no pacote Go nativo (`main.go` ou `harness/`).
2. Não adicionar dependências externas em Go sem aprovação explícita (manter a biblioteca padrão do Go `encoding/json`, `flag`, `os`, `path/filepath`, `regexp`, `strings` como prioridade máxima).
3. Todas as alterações devem manter conformidade com os critérios descritos em [`AGENTS.md`](./AGENTS.md).
