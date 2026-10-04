package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ========================================================
// ESTRUTURAS DO DUMP DO BUBBLE
// ========================================================

type BubbleAppDump struct {
	Pages         map[string]BubblePageDump  `json:"pages"`
	CustomOptions json.RawMessage            `json:"custom_options"`
	OptionSets    map[string]OptionSetDump   `json:"option_sets"`
	APIConnectors json.RawMessage            `json:"api_connectors"`
	UserTypes     map[string]UserTypeDump    `json:"user_types"`
	Extra         map[string]json.RawMessage `json:"-"`
}

type BubblePageDump struct {
	Name         string                     `json:"name"`
	Type         string                     `json:"type"`
	Properties   map[string]interface{}     `json:"properties"`
	Elements     map[string]json.RawMessage `json:"elements"`
	Workflows    map[string]json.RawMessage `json:"workflows"`
	CustomStates map[string]interface{}     `json:"custom_states"`
}

type OptionSetDump struct {
	Display string                   `json:"display"`
	Deleted bool                     `json:"deleted"`
	Values  map[string]OptionValDump `json:"values"`
}

type OptionValDump struct {
	Display    string `json:"display"`
	DbValue    string `json:"db_value"`
	SortFactor int    `json:"sort_factor"`
}

type UserTypeDump struct {
	Display string                   `json:"display"`
	Fields  map[string]UserFieldDump `json:"fields"`
}

type UserFieldDump struct {
	Display string      `json:"display"`
	Value   interface{} `json:"value"`
	Deleted bool        `json:"deleted"`
}

type ManifestDump struct {
	GeneratedAt time.Time      `json:"generated_at"`
	SourceFile  string         `json:"source_file"`
	TotalPages  int            `json:"total_pages"`
	TablesFound []string       `json:"tables_found"`
	Pages       []PageManifest `json:"pages"`
}

type PageManifest struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	ElementsCount  int    `json:"elements_count"`
	WorkflowsCount int    `json:"workflows_count"`
	ElementsPath   string `json:"elements_path"`
	WorkflowsPath  string `json:"workflows_path,omitempty"`
}

type BubbleElementNode struct {
	Type       string                       `json:"type"`
	Properties map[string]interface{}       `json:"properties"`
	Elements   map[string]BubbleElementNode `json:"elements"`
}

type BubbleWorkflowNode struct {
	Type       string                      `json:"type"`
	ID         string                      `json:"id"`
	Properties map[string]interface{}     `json:"properties"`
	Actions    map[string]BubbleActionNode `json:"actions"`
}

type BubbleActionNode struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

type FlattenedElement struct {
	ID    string
	Type  string
	Depth int
	Text  string
	Props map[string]interface{}
}

type TypeMapping struct {
	Postgres   string
	TypeScript string
	IsRelation bool
	RelTable   string
	IsOption   bool
	OptionName string
	IsList     bool
}

// ========================================================
// HELPERS
// ========================================================

func saveJSON(filePath string, data interface{}) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func sanitizeIdentifier(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	res := re.ReplaceAllString(s, "_")
	if len(res) == 0 {
		return "field"
	}
	return res
}

func capitalize(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func cleanColumnName(rawKey string) string {
	rePrefix := regexp.MustCompile(`^[0-9]+__|^[a-zA-Z]__`)
	res := rePrefix.ReplaceAllString(rawKey, "")
	for _, suffix := range []string{"_text", "_date", "_boolean", "_number", "_file", "_image"} {
		res = strings.TrimSuffix(res, suffix)
	}
	reNonAlpha := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	res = strings.ToLower(reNonAlpha.ReplaceAllString(res, "_"))
	if res == "" {
		res = strings.ToLower(rawKey)
	}
	return res
}

func mapType(valStr string, colName string) TypeMapping {
	valStr = strings.TrimSpace(valStr)
	switch {
	case valStr == "text":
		return TypeMapping{Postgres: "TEXT", TypeScript: "string"}
	case valStr == "number":
		lowerCol := strings.ToLower(colName)
		if strings.Contains(lowerCol, "ano") || strings.Contains(lowerCol, "count") || strings.Contains(lowerCol, "qtd") {
			return TypeMapping{Postgres: "INTEGER", TypeScript: "number"}
		}
		return TypeMapping{Postgres: "NUMERIC(15, 2)", TypeScript: "number"}
	case valStr == "boolean":
		return TypeMapping{Postgres: "BOOLEAN DEFAULT FALSE", TypeScript: "boolean"}
	case valStr == "date":
		return TypeMapping{Postgres: "TIMESTAMPTZ", TypeScript: "Date | string"}
	case valStr == "file" || valStr == "image":
		return TypeMapping{Postgres: "TEXT", TypeScript: "string"}
	case valStr == "user" || valStr == "user1":
		return TypeMapping{Postgres: "UUID", TypeScript: "string", IsRelation: true, RelTable: "users"}
	case strings.HasPrefix(valStr, "custom."):
		target := strings.TrimPrefix(valStr, "custom.")
		if target == "user" || target == "user1" {
			target = "users"
		}
		return TypeMapping{Postgres: "UUID", TypeScript: "string", IsRelation: true, RelTable: target}
	case strings.HasPrefix(valStr, "option."):
		opt := strings.TrimPrefix(valStr, "option.")
		return TypeMapping{Postgres: "VARCHAR(100)", TypeScript: "string", IsOption: true, OptionName: opt}
	case strings.HasPrefix(valStr, "list."):
		inner := strings.TrimPrefix(valStr, "list.")
		sub := mapType(inner, colName)
		return TypeMapping{Postgres: sub.Postgres + "[]", TypeScript: sub.TypeScript + "[]", IsList: true, IsRelation: sub.IsRelation, RelTable: sub.RelTable}
	default:
		return TypeMapping{Postgres: "TEXT", TypeScript: "string"}
	}
}

// ========================================================
// 1. DECOMPILADOR DO BUBBLE
// ========================================================

func runDecompiler(inputFile, outputDir string) (*BubbleAppDump, *ManifestDump, error) {
	fmt.Printf("⚡ [Bubble Decompiler] Extraindo dump de: %s\n", inputFile)

	file, err := os.Open(inputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	rawBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler buffer: %w", err)
	}

	var dump BubbleAppDump
	if err := json.Unmarshal(rawBytes, &dump); err != nil {
		return nil, nil, fmt.Errorf("erro no parsing JSON do Bubble: %w", err)
	}

	// 1. Banco de Dados e Enums
	fmt.Println("📦 Fatiando Camada 1: Banco de Dados & Enums...")
	dbPath := filepath.Join(outputDir, "01_database")

	if len(dump.OptionSets) > 0 {
		saveJSON(filepath.Join(dbPath, "option_sets.json"), dump.OptionSets)
	} else if len(dump.CustomOptions) > 0 {
		saveJSON(filepath.Join(dbPath, "option_sets.json"), dump.CustomOptions)
	}

	reCustomTypes := regexp.MustCompile(`custom\.[a-zA-Z0-9_]+`)
	matches := reCustomTypes.FindAllString(string(rawBytes), -1)
	tableSet := make(map[string]bool)
	for _, m := range matches {
		tableSet[m] = true
	}
	tables := make([]string, 0, len(tableSet))
	for t := range tableSet {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	saveJSON(filepath.Join(dbPath, "detected_tables.json"), map[string]interface{}{
		"total":  len(tables),
		"tables": tables,
	})

	// 2. Telas, Elementos e Workflows
	fmt.Println("🖥️  Fatiando Camada 2 & 3: Telas, Elementos e Workflows...")
	manifestPages := make([]PageManifest, 0, len(dump.Pages))

	for pageID, page := range dump.Pages {
		name := page.Name
		if strings.TrimSpace(name) == "" {
			name = pageID
		}

		pageDir := filepath.Join(outputDir, "03_ui_pages", name)
		elementsFile := filepath.Join(pageDir, "elements_tree.json")
		metaFile := filepath.Join(pageDir, "page_meta.json")

		saveJSON(elementsFile, page.Elements)
		saveJSON(metaFile, map[string]interface{}{
			"id":            pageID,
			"name":          page.Name,
			"properties":    page.Properties,
			"custom_states": page.CustomStates,
		})

		wfPathRel := ""
		if len(page.Workflows) > 0 {
			wfDir := filepath.Join(outputDir, "02_workflows", "pages")
			wfFile := filepath.Join(wfDir, fmt.Sprintf("%s_workflows.json", name))
			saveJSON(wfFile, page.Workflows)
			wfPathRel = filepath.ToSlash(filepath.Join("02_workflows", "pages", fmt.Sprintf("%s_workflows.json", name)))
		}

		manifestPages = append(manifestPages, PageManifest{
			Name:           name,
			ID:             pageID,
			ElementsCount:  len(page.Elements),
			WorkflowsCount: len(page.Workflows),
			ElementsPath:   filepath.ToSlash(filepath.Join("03_ui_pages", name, "elements_tree.json")),
			WorkflowsPath:  wfPathRel,
		})
	}

	// 3. Integrações (API Connectors)
	if len(dump.APIConnectors) > 0 {
		fmt.Println("🔌 Fatiando Camada 4: Integrações e API Connector...")
		saveJSON(filepath.Join(outputDir, "04_integrations", "api_connectors.json"), dump.APIConnectors)
	}

	// 4. Manifesto
	manifest := ManifestDump{
		GeneratedAt: time.Now().UTC(),
		SourceFile:  inputFile,
		TotalPages:  len(dump.Pages),
		TablesFound: tables,
		Pages:       manifestPages,
	}
	saveJSON(filepath.Join(outputDir, "app_manifest.json"), manifest)

	return &dump, &manifest, nil
}

// ========================================================
// 2. GERADOR DE ESPECIFICAÇÕES (SPEC.MD)
// ========================================================

func runSpecGenerator(harnessDir string, app *BubbleAppDump, manifest *ManifestDump) {
	fmt.Println("📊 Gerando 01_database/SPEC.md...")
	generateDatabaseSpec(harnessDir, app)

	fmt.Println("⚡ Gerando 02_workflows/pages/ SPEC.md por tela...")
	generateWorkflowSpecs(harnessDir, manifest)

	fmt.Println("🎨 Gerando 03_ui_pages/ SPEC.md por tela...")
	generateUIPagesSpecs(harnessDir, manifest)
}

func generateDatabaseSpec(harnessDir string, app *BubbleAppDump) {
	var sb strings.Builder

	sb.WriteString("# Especificação Técnica do Banco de Dados (Database Spec)\n\n")
	sb.WriteString("**Aplicação:** Bubble to Code Engine\n")
	sb.WriteString(fmt.Sprintf("**Data de Geração:** %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("**Engine:** Go Native High-Performance Spec Engine\n\n")
	sb.WriteString("---\n\n## 1. Visão Geral da Arquitetura de Dados\n\n")
	sb.WriteString(fmt.Sprintf("O banco de dados original contém **%d entidades principais (User Types)** e **%d Option Sets (Enums/Domínios Fixos)**.\n\n", len(app.UserTypes), len(app.OptionSets)))
	sb.WriteString("---\n\n## 2. Option Sets (Enums e Tabelas de Domínio)\n\n")
	sb.WriteString("| Option Set | Nome Amigável | Valores Identificados |\n| :--- | :--- | :--- |\n")

	optKeys := make([]string, 0, len(app.OptionSets))
	for k := range app.OptionSets {
		optKeys = append(optKeys, k)
	}
	sort.Strings(optKeys)

	for _, k := range optKeys {
		opt := app.OptionSets[k]
		if opt.Deleted {
			continue
		}
		var valNames []string
		for _, v := range opt.Values {
			disp := v.Display
			if disp == "" {
				disp = v.DbValue
			}
			if disp != "" {
				valNames = append(valNames, "`"+disp+"`")
			}
		}
		joinedVals := strings.Join(valNames, ", ")
		if joinedVals == "" {
			joinedVals = "Nenhum valor ativo"
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", k, opt.Display, joinedVals))
	}

	sb.WriteString("\n---\n\n## 3. Dicionário de Dados e Entidades\n\n")

	var ddlTables []string
	var ddlForeignKeys []string
	var ddlIndices []string

	tblKeys := make([]string, 0, len(app.UserTypes))
	for k := range app.UserTypes {
		tblKeys = append(tblKeys, k)
	}
	sort.Strings(tblKeys)

	for _, tblKey := range tblKeys {
		tblData := app.UserTypes[tblKey]
		tableName := tblKey
		if tableName == "user" || tableName == "user1" {
			tableName = "users"
		}
		displayName := tblData.Display
		if displayName == "" {
			displayName = tableName
		}

		sb.WriteString(fmt.Sprintf("### Entidade: `%s` (%s)\n\n", tableName, displayName))
		sb.WriteString("| Coluna (PostgreSQL) | Campo Original Bubble | Tipo Bubble | Tipo PostgreSQL | Tipo TypeScript | Relacionamento / Notas |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
		sb.WriteString("| `id` | (Bubble ID) | unique id | `UUID PRIMARY KEY` | `string` | Chave primária gerada via gen_random_uuid() |\n")
		sb.WriteString("| `created_at` | Creation Date | date | `TIMESTAMPTZ NOT NULL` | `Date` | Data de criação |\n")
		sb.WriteString("| `updated_at` | Modified Date | date | `TIMESTAMPTZ NOT NULL` | `Date` | Última alteração |\n")

		var tableCols []string
		tableCols = append(tableCols, "  id UUID PRIMARY KEY DEFAULT gen_random_uuid()")
		tableCols = append(tableCols, "  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
		tableCols = append(tableCols, "  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")

		fieldKeys := make([]string, 0, len(tblData.Fields))
		for f := range tblData.Fields {
			fieldKeys = append(fieldKeys, f)
		}
		sort.Strings(fieldKeys)

		for _, fKey := range fieldKeys {
			fData := tblData.Fields[fKey]
			if fData.Deleted {
				continue
			}
			colName := cleanColumnName(fKey)
			typeStr := fmt.Sprintf("%v", fData.Value)
			mapped := mapType(typeStr, colName)

			notes := "-"
			if mapped.IsRelation && mapped.RelTable != "" {
				notes = fmt.Sprintf("FK -> `%s`", mapped.RelTable)
			} else if mapped.IsOption {
				notes = fmt.Sprintf("Option Set: `%s`", mapped.OptionName)
			} else if mapped.IsList {
				notes = "Lista / Array de valores"
			}

			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | `%s` | `%s` | %s |\n", colName, fData.Display, typeStr, mapped.Postgres, mapped.TypeScript, notes))
			tableCols = append(tableCols, fmt.Sprintf("  %s %s", colName, mapped.Postgres))

			if mapped.IsRelation && mapped.RelTable != "" && !mapped.IsList {
				target := mapped.RelTable
				if target == "user" || target == "user1" {
					target = "users"
				}
				ddlForeignKeys = append(ddlForeignKeys, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(id) ON DELETE SET NULL;", tableName, tableName, colName, colName, target))
				ddlIndices = append(ddlIndices, fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s(%s);", tableName, colName, tableName, colName))
			}
		}

		ddlTables = append(ddlTables, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", tableName, strings.Join(tableCols, ",\n")))
		ddlIndices = append(ddlIndices, fmt.Sprintf("CREATE INDEX idx_%s_created_at ON %s(created_at);", tableName, tableName))
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n## 4. Script DDL em PostgreSQL\n\n```sql\n")
	sb.WriteString("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\nCREATE EXTENSION IF NOT EXISTS \"pgcrypto\";\n\n")
	sb.WriteString("-- ========================================================\n-- 1. CRIAÇÃO DAS TABELAS\n-- ========================================================\n")
	sb.WriteString(strings.Join(ddlTables, "\n\n"))
	sb.WriteString("\n\n-- ========================================================\n-- 2. CHAVES ESTRANGEIRAS (FOREIGN KEYS)\n-- ========================================================\n")
	sb.WriteString(strings.Join(ddlForeignKeys, "\n"))
	sb.WriteString("\n\n-- ========================================================\n-- 3. ÍNDICES DE PERFORMANCE\n-- ========================================================\n")
	sb.WriteString(strings.Join(ddlIndices, "\n"))
	sb.WriteString("\n```\n\n")

	sb.WriteString("---\n\n## 5. Mapeamento de Tipos Consolidado\n\n")
	sb.WriteString("| Bubble Type | PostgreSQL Type | TypeScript Type | Estratégia de Migração |\n| :--- | :--- | :--- | :--- |\n")
	sb.WriteString("| `text` | `TEXT` | `string` | UTF-8 nativo |\n")
	sb.WriteString("| `number` | `NUMERIC(15,2)` / `INTEGER` | `number` | Campos monetários usam NUMERIC, contadores usam INTEGER |\n")
	sb.WriteString("| `boolean` | `BOOLEAN DEFAULT FALSE` | `boolean` | Default explícito false |\n")
	sb.WriteString("| `date` | `TIMESTAMPTZ` | `Date` / `string` | Salvar UTC com fuso horário |\n")
	sb.WriteString("| `file` / `image` | `TEXT` | `string` | Armazenar URL Cloud Storage (S3 / Supabase Storage) |\n")
	sb.WriteString("| `user` | `UUID` | `string` | Foreign key para tabela `users` (Auth) |\n")
	sb.WriteString("| `custom.<type>` | `UUID` | `string` | Foreign key com integridade referencial `ON DELETE SET NULL` |\n")
	sb.WriteString("| `option.<type>` | `VARCHAR(100)` | `string` | Armazenar código canônico do Option Set |\n")
	sb.WriteString("| `list.<type>` | `TYPE[]` / JSONB | `Array<T>` | Array nativo do PostgreSQL ou tabela associativa |\n")

	outPath := filepath.Join(harnessDir, "01_database", "SPEC.md")
	os.WriteFile(outPath, []byte(sb.String()), 0644)
}

func generateWorkflowSpecs(harnessDir string, manifest *ManifestDump) {
	wfBaseDir := filepath.Join(harnessDir, "02_workflows", "pages")
	var globalIndex strings.Builder

	globalIndex.WriteString("# Índice Consolidado de Workflows da Aplicação\n\n")
	globalIndex.WriteString("| Página | Quantidade de Workflows | Arquivo JSON | Especificação (SPEC.md) |\n| :--- | :--- | :--- | :--- |\n")

	for _, page := range manifest.Pages {
		jsonFile := filepath.Join(wfBaseDir, page.Name+"_workflows.json")
		subFolder := filepath.Join(wfBaseDir, page.Name)
		os.MkdirAll(subFolder, 0755)

		var pageWorkflows map[string]BubbleWorkflowNode
		wfCount := 0

		if raw, err := os.ReadFile(jsonFile); err == nil {
			json.Unmarshal(raw, &pageWorkflows)
			wfCount = len(pageWorkflows)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Especificação de Workflows e Regras de Negócio: Página `%s`\n\n", page.Name))
		sb.WriteString(fmt.Sprintf("**Página:** `%s`  \n", page.Name))
		sb.WriteString(fmt.Sprintf("**Total de Workflows Identificados:** %d  \n", wfCount))
		sb.WriteString(fmt.Sprintf("**Arquivo Fonte:** `02_workflows/pages/%s_workflows.json`  \n\n", page.Name))
		sb.WriteString("---\n\n## 1. Resumo Executivo da Página\n\n")

		if wfCount == 0 {
			sb.WriteString(fmt.Sprintf("A página `%s` é estática e não possui workflows ou regras de negócio dinâmicas registradas no Bubble.\n\n", page.Name))
			sb.WriteString("## 2. Casos de Uso (Services) em TypeScript\n\n```typescript\nexport interface StaticPageProps {\n  redirectTo?: string;\n}\n```\n")
		} else {
			sb.WriteString(fmt.Sprintf("A página `%s` possui **%d workflows** registrados no Bubble cobrindo gatilhos de UI, manipulação de dados e regras de negócio.\n\n", page.Name, wfCount))
			sb.WriteString("## 2. Tabela Detalhada de Gatilhos e Ações\n\n")
			sb.WriteString("| ID do Workflow | Gatilho (Trigger) | Elemento Alvo | Condição (`Only when`) | Ações Executadas |\n| :--- | :--- | :--- | :--- | :--- |\n")

			wfIDs := make([]string, 0, len(pageWorkflows))
			for wid := range pageWorkflows {
				wfIDs = append(wfIDs, wid)
			}
			sort.Strings(wfIDs)

			for _, wid := range wfIDs {
				wf := pageWorkflows[wid]
				elemID := "Página / Global"
				if eid, ok := wf.Properties["element_id"].(string); ok && eid != "" {
					elemID = eid
				}
				condStr := "Sem restrição (Sempre executa)"
				if c, ok := wf.Properties["condition"]; ok && c != nil {
					if cJson, err := json.Marshal(c); err == nil {
						condStr = strings.ReplaceAll(string(cJson), "|", "\\|")
					}
				}

				var actDescs []string
				actIdxs := make([]string, 0, len(wf.Actions))
				for aidx := range wf.Actions {
					actIdxs = append(actIdxs, aidx)
				}
				sort.Strings(actIdxs)

				for i, aidx := range actIdxs {
					act := wf.Actions[aidx]
					desc := act.Type
					switch act.Type {
					case "SetCustomState":
						desc = fmt.Sprintf("SetCustomState: `%v` = `%v`", act.Properties["custom_state"], act.Properties["value"])
					case "CreateThing":
						desc = fmt.Sprintf("Criar `%v`", act.Properties["type"])
					case "ChangeThing":
						desc = fmt.Sprintf("Atualizar `%v`", act.Properties["type"])
					case "NavigateToPage":
						desc = fmt.Sprintf("Navegar para `%v`", act.Properties["page"])
					}
					actDescs = append(actDescs, fmt.Sprintf("%d. %s", i+1, desc))
				}
				actsFormatted := strings.Join(actDescs, "<br/>")
				if actsFormatted == "" {
					actsFormatted = "Nenhuma ação"
				}

				sb.WriteString(fmt.Sprintf("| `%s` | **%s** | `%s` | `%s` | %s |\n", wid, wf.Type, elemID, condStr, actsFormatted))
			}

			sb.WriteString("\n---\n\n## 3. Proposta de Casos de Uso (Services) em TypeScript\n\n```typescript\n")
			sb.WriteString("import { z } from 'zod';\n\n")
			sb.WriteString(fmt.Sprintf("export const %sInputSchema = z.object({\n  action: z.string(),\n  payload: z.record(z.any()).optional()\n});\n\n", capitalize(page.Name)))
			sb.WriteString(fmt.Sprintf("export type %sInput = z.infer<typeof %sInputSchema>;\n\n", capitalize(page.Name), capitalize(page.Name)))
			sb.WriteString(fmt.Sprintf("export class %sService {\n", capitalize(page.Name)))
			sb.WriteString(fmt.Sprintf("  async handleAction(input: %sInput): Promise<{ success: boolean; data?: any; error?: string }> {\n", capitalize(page.Name)))
			sb.WriteString("    return { success: true, data: { status: 'handled', action: input.action } };\n  }\n}\n```\n")
		}

		os.WriteFile(filepath.Join(subFolder, "SPEC.md"), []byte(sb.String()), 0644)
		os.WriteFile(filepath.Join(wfBaseDir, page.Name+"_SPEC.md"), []byte(sb.String()), 0644)
		globalIndex.WriteString(fmt.Sprintf("| `%s` | %d | [`%s_workflows.json`](./%s_workflows.json) | [`SPEC.md`](./%s/SPEC.md) |\n", page.Name, wfCount, page.Name, page.Name, page.Name))
	}

	os.WriteFile(filepath.Join(wfBaseDir, "SPEC.md"), []byte(globalIndex.String()), 0644)
}

func flattenElements(nodeMap map[string]json.RawMessage, depth int, result *[]FlattenedElement) {
	for k, raw := range nodeMap {
		var node BubbleElementNode
		if err := json.Unmarshal(raw, &node); err != nil {
			continue
		}
		typ := node.Type
		if typ == "" {
			if pt, ok := node.Properties["type"].(string); ok {
				typ = pt
			} else {
				typ = "Element"
			}
		}
		label := ""
		if t, ok := node.Properties["text"].(string); ok {
			label = t
		} else if p, ok := node.Properties["placeholder"].(string); ok {
			label = p
		}

		*result = append(*result, FlattenedElement{
			ID:    k,
			Type:  typ,
			Depth: depth,
			Text:  label,
			Props: node.Properties,
		})

		if len(node.Elements) > 0 {
			var rawSub = make(map[string]json.RawMessage)
			for subK, subV := range node.Elements {
				if b, err := json.Marshal(subV); err == nil {
					rawSub[subK] = b
				}
			}
			flattenElements(rawSub, depth+1, result)
		}
	}
}

func generateUIPagesSpecs(harnessDir string, manifest *ManifestDump) {
	uiBaseDir := filepath.Join(harnessDir, "03_ui_pages")

	for _, page := range manifest.Pages {
		pageFolder := filepath.Join(uiBaseDir, page.Name)
		treeFile := filepath.Join(pageFolder, "elements_tree.json")

		var rawMap map[string]json.RawMessage
		var flatList []FlattenedElement

		if bytes, err := os.ReadFile(treeFile); err == nil {
			json.Unmarshal(bytes, &rawMap)
			flattenElements(rawMap, 0, &flatList)
		}

		var inputs, tables, buttons []FlattenedElement
		reInput := regexp.MustCompile(`(?i)input|text|dropdown|multiselect|checkbox|radio|upload`)
		reTable := regexp.MustCompile(`(?i)repeatinggroup|table|list|grid`)
		reButton := regexp.MustCompile(`(?i)button`)

		for _, item := range flatList {
			if reInput.MatchString(item.Type) || item.Props["placeholder"] != nil {
				inputs = append(inputs, item)
			}
			if reTable.MatchString(item.Type) {
				tables = append(tables, item)
			}
			if reButton.MatchString(item.Type) {
				buttons = append(buttons, item)
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Especificação Visual e Componentes: Página `%s`\n\n", page.Name))
		sb.WriteString(fmt.Sprintf("**Página:** `%s`  \n", page.Name))
		sb.WriteString(fmt.Sprintf("**Total de Nós da Árvore de UI:** %d  \n", len(flatList)))
		sb.WriteString(fmt.Sprintf("**Arquivo Fonte:** `03_ui_pages/%s/elements_tree.json`  \n\n", page.Name))
		sb.WriteString("---\n\n## 1. Visão Geral da Interface\n\n")
		sb.WriteString(fmt.Sprintf("A interface de `%s` é composta por **%d elementos estruturais**.\n", page.Name, len(flatList)))
		sb.WriteString(fmt.Sprintf("- **Campos de Entrada/Formulários:** %d identificados\n", len(inputs)))
		sb.WriteString(fmt.Sprintf("- **Tabelas / Repeating Groups:** %d identificados\n", len(tables)))
		sb.WriteString(fmt.Sprintf("- **Botões e Ações:** %d identificados\n\n", len(buttons)))

		sb.WriteString("---\n\n## 2. Hierarquia Visual de Componentes (Tree View)\n\n```yaml\n")
		sb.WriteString(fmt.Sprintf("Page: %s\n", page.Name))
		limit := len(flatList)
		if limit > 40 {
			limit = 40
		}
		for _, el := range flatList[:limit] {
			indent := strings.Repeat("  ", el.Depth+1)
			sb.WriteString(fmt.Sprintf("%s- id: \"%s\"\n%s  type: \"%s\"\n", indent, el.ID, indent, el.Type))
			if el.Text != "" {
				cleanText := strings.ReplaceAll(el.Text, "\"", "'")
				if len(cleanText) > 50 {
					cleanText = cleanText[:50]
				}
				sb.WriteString(fmt.Sprintf("%s  label: \"%s\"\n", indent, cleanText))
			}
		}
		sb.WriteString("```\n\n---\n\n## 3. Elementos de Formulário e Controles de Entrada\n\n")
		sb.WriteString("| ID do Elemento | Tipo no Bubble | Rótulo / Placeholder | Tipo de Dado Estimado |\n| :--- | :--- | :--- | :--- |\n")
		if len(inputs) == 0 {
			sb.WriteString("| - | Nenhum formulário complexo | - | - |\n")
		} else {
			inLimit := len(inputs)
			if inLimit > 25 {
				inLimit = 25
			}
			for _, in := range inputs[:inLimit] {
				disp := in.Text
				if disp == "" {
					disp = "-"
				} else if len(disp) > 40 {
					disp = disp[:40]
				}
				sb.WriteString(fmt.Sprintf("| `%s` | `%s` | \"%s\" | `string` |\n", in.ID, in.Type, strings.ReplaceAll(disp, "\"", "'")))
			}
		}

		sb.WriteString("\n---\n\n## 4. Tabelas, Listas e Repeating Groups\n\n")
		sb.WriteString("| ID do Container | Tipo | Fonte de Dados (Data Source) | Propósito Estimado |\n| :--- | :--- | :--- | :--- |\n")
		if len(tables) == 0 {
			sb.WriteString("| - | Nenhuma tabela de repetição | - | Interface estática / Formulário direto |\n")
		} else {
			for _, t := range tables {
				ds, _ := json.Marshal(t.Props["data_source"])
				sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | Listagem e Repetição |\n", t.ID, t.Type, string(ds)))
			}
		}

		sb.WriteString("\n---\n\n## 5. Implementação Equivalente em React + Tailwind CSS\n\n```tsx\n")
		sb.WriteString("import React, { useState } from 'react';\n")
		sb.WriteString("import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';\n")
		sb.WriteString("import { Button } from '@/components/ui/button';\n")
		sb.WriteString("import { Input } from '@/components/ui/input';\n")
		sb.WriteString("import { Label } from '@/components/ui/label';\n\n")
		sb.WriteString(fmt.Sprintf("export const %sPage: React.FC = () => {\n", capitalize(page.Name)))
		sb.WriteString("  const [formData, setFormData] = useState<Record<string, string>>({});\n\n")
		sb.WriteString("  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {\n")
		sb.WriteString("    setFormData(prev => ({ ...prev, [e.target.name]: e.target.value }));\n  };\n\n")
		sb.WriteString("  return (\n    <div className=\"min-h-screen bg-slate-50 dark:bg-slate-900 p-6 md:p-12 text-slate-800 dark:text-slate-100\">\n")
		sb.WriteString("      <div className=\"max-w-4xl mx-auto space-y-8\">\n")
		sb.WriteString("        <header className=\"border-b border-slate-200 dark:border-slate-800 pb-4\">\n")
		sb.WriteString(fmt.Sprintf("          <h1 className=\"text-3xl font-bold tracking-tight text-slate-900 dark:text-white capitalize\">%s</h1>\n", strings.ReplaceAll(page.Name, "_", " ")))
		sb.WriteString("          <p className=\"text-sm text-slate-500 mt-1\">Componente gerado a partir da árvore visual do Bubble.</p>\n")
		sb.WriteString("        </header>\n")
		sb.WriteString("        <Card className=\"shadow-lg bg-white dark:bg-slate-800\">\n")
		sb.WriteString("          <CardHeader><CardTitle className=\"text-xl font-semibold\">Formulário</CardTitle></CardHeader>\n")
		sb.WriteString("          <CardContent>\n            <form className=\"space-y-6\">\n              <div className=\"grid grid-cols-1 md:grid-cols-2 gap-6\">\n")

		fieldLimit := len(inputs)
		if fieldLimit > 6 {
			fieldLimit = 6
		}
		for _, in := range inputs[:fieldLimit] {
			label := in.Text
			if label == "" {
				label = sanitizeIdentifier(in.ID)
			}
			cleanId := sanitizeIdentifier(in.ID)
			sb.WriteString(fmt.Sprintf("                <div className=\"space-y-2\">\n                  <Label htmlFor=\"%s\">%s</Label>\n                  <Input id=\"%s\" name=\"%s\" onChange={handleChange} />\n                </div>\n", cleanId, label, cleanId, cleanId))
		}
		sb.WriteString("              </div>\n              <div className=\"flex justify-end gap-3 pt-4 border-t\">\n")
		sb.WriteString("                <Button type=\"button\" variant=\"outline\">Cancelar</Button>\n")
		sb.WriteString("                <Button type=\"submit\" className=\"bg-blue-600 text-white\">Salvar</Button>\n")
		sb.WriteString("              </div>\n            </form>\n          </CardContent>\n        </Card>\n      </div>\n    </div>\n  );\n};\n")
		sb.WriteString(fmt.Sprintf("export default %sPage;\n```\n", capitalize(page.Name)))

		os.WriteFile(filepath.Join(pageFolder, "SPEC.md"), []byte(sb.String()), 0644)
	}
}

// ========================================================
// 3. MAIN: CLI UNIFICADA
// ========================================================

func main() {
	inputFile := flag.String("file", "app.bubble", "Arquivo .bubble a ser processado")
	outputDir := flag.String("out", "extracted_app_harness", "Diretório de saída")
	decompileOnly := flag.Bool("decompile", false, "Executa somente a extração JSON do dump")
	specsOnly := flag.Bool("specs", false, "Executa somente a geração dos SPEC.md")
	flag.Parse()

	if len(flag.Args()) > 0 && *inputFile == "app.bubble" {
		*inputFile = flag.Args()[0]
	}

	start := time.Now()
	fmt.Println("🚀 [bubble-to-code] Motor de Engenharia Reversa Nativo")

	if !*specsOnly {
		dump, manifest, err := runDecompiler(*inputFile, *outputDir)
		if err != nil {
			fmt.Printf("❌ Erro no decompiler: %v\n", err)
			os.Exit(1)
		}
		if !*decompileOnly {
			runSpecGenerator(*outputDir, dump, manifest)
		}
	} else {
		rawBytes, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Printf("❌ Erro ao ler %s: %v\n", *inputFile, err)
			os.Exit(1)
		}
		var app BubbleAppDump
		if err := json.Unmarshal(rawBytes, &app); err != nil {
			fmt.Printf("❌ Erro ao parsear dump: %v\n", err)
			os.Exit(1)
		}
		manifestBytes, err := os.ReadFile(filepath.Join(*outputDir, "app_manifest.json"))
		if err != nil {
			fmt.Printf("❌ Erro ao ler manifesto: %v\n", err)
			os.Exit(1)
		}
		var manifest ManifestDump
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			fmt.Printf("❌ Erro ao parsear manifesto: %v\n", err)
			os.Exit(1)
		}
		runSpecGenerator(*outputDir, &app, &manifest)
	}

	fmt.Printf("\n✨ Pipeline concluído com sucesso em %v!\n", time.Since(start))
	fmt.Printf("📁 Artefatos prontos em: ./%s/\n", *outputDir)
}