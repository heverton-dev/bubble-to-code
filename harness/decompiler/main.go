package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Estrutura do arquivo de dump do Bubble.io
type BubbleDump struct {
	Pages         map[string]BubblePage      `json:"pages"`
	CustomOptions json.RawMessage            `json:"custom_options"`
	OptionSets    json.RawMessage            `json:"option_sets"`
	APIConnectors json.RawMessage            `json:"api_connectors"`
	UserTypes     json.RawMessage            `json:"user_types"`
	Extra         map[string]json.RawMessage `json:"-"`
}

type BubblePage struct {
	Name         string                     `json:"name"`
	Type         string                     `json:"type"`
	Properties   map[string]interface{}     `json:"properties"`
	Elements     map[string]json.RawMessage `json:"elements"`
	Workflows    map[string]json.RawMessage `json:"workflows"`
	CustomStates map[string]interface{}     `json:"custom_states"`
}

// Manifesto estruturado para os harnesses de IA
type Manifest struct {
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

func main() {
	inputFile := flag.String("file", "gestao-contratos-conexao.bubble", "Caminho do arquivo .bubble")
	outputDir := flag.String("out", "extracted_app_harness", "Diretório de saída para os artefatos")
	flag.Parse()

	if len(flag.Args()) > 0 && *inputFile == "gestao-contratos-conexao.bubble" {
		*inputFile = flag.Args()[0]
	}

	fmt.Printf("⚡ [Bubble Decompiler] Iniciando extração de: %s\n", *inputFile)
	start := time.Now()

	file, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("❌ Erro ao abrir arquivo: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	rawBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("❌ Erro ao ler buffer do arquivo: %v\n", err)
		os.Exit(1)
	}

	var dump BubbleDump
	if err := json.Unmarshal(rawBytes, &dump); err != nil {
		fmt.Printf("❌ Erro no parsing JSON do Bubble: %v\n", err)
		os.Exit(1)
	}

	// 1. CAMADA DE BANCO DE DADOS (Option Sets & Tabelas Customizadas)
	fmt.Println("📦 Fatiando Camada 1: Banco de Dados & Enums...")
	dbPath := filepath.Join(*outputDir, "01_database")

	if len(dump.OptionSets) > 0 {
		saveJSON(filepath.Join(dbPath, "option_sets.json"), dump.OptionSets)
	} else if len(dump.CustomOptions) > 0 {
		saveJSON(filepath.Join(dbPath, "option_sets.json"), dump.CustomOptions)
	}

	// Identificação de tabelas customizadas (custom.*)
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
	saveJSON(filepath.Join(dbPath, "detected_tables.json"), map[string]interface{}{
		"total":  len(tables),
		"tables": tables,
	})

	// 2. CAMADA DE WORKFLOWS E INTERFACE POR PÁGINA
	fmt.Println("🖥️  Fatiando Camada 2 & 3: Telas, Elementos e Workflows...")
	manifestPages := make([]PageManifest, 0, len(dump.Pages))

	for pageID, page := range dump.Pages {
		name := page.Name
		if strings.TrimSpace(name) == "" {
			name = pageID
		}

		pageDir := filepath.Join(*outputDir, "03_ui_pages", name)
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
			wfDir := filepath.Join(*outputDir, "02_workflows", "pages")
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

	// 3. CAMADA DE INTEGRAÇÕES (API Connector)
	if len(dump.APIConnectors) > 0 {
		fmt.Println("🔌 Fatiando Camada 4: Integrações e API Connector...")
		saveJSON(filepath.Join(*outputDir, "04_integrations", "api_connectors.json"), dump.APIConnectors)
	}

	// 4. MANIFESTO PARA HARNESSES DE IA
	manifest := Manifest{
		GeneratedAt: time.Now().UTC(),
		SourceFile:  *inputFile,
		TotalPages:  len(dump.Pages),
		TablesFound: tables,
		Pages:       manifestPages,
	}
	saveJSON(filepath.Join(*outputDir, "app_manifest.json"), manifest)

	fmt.Printf("\n✨ Concluído com sucesso em %v!\n", time.Since(start))
	fmt.Printf("📁 Artefatos prontos em: ./%s/\n", *outputDir)
	fmt.Printf("📄 Manifesto gerado em: ./%s/app_manifest.json\n", *outputDir)
}
