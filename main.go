package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Configurações do Ollama Local
const (
	OllamaEndpoint = "http://localhost:11434/api/generate"
	OllamaModel    = "llama3.1:8b"
)

type BubbleApp struct {
	Pages         map[string]BubblePage `json:"pages"`
	CustomOptions json.RawMessage       `json:"custom_options"`
	OptionSets    json.RawMessage       `json:"option_sets"`
	APIConnectors json.RawMessage       `json:"api_connectors"`
	UserTypes     json.RawMessage       `json:"user_types"`
}

type BubblePage struct {
	Name         string                     `json:"name"`
	Type         string                     `json:"type"`
	Properties   map[string]interface{}     `json:"properties"`
	Elements     map[string]json.RawMessage `json:"elements"`
	Workflows    map[string]json.RawMessage `json:"workflows"`
	CustomStates map[string]interface{}     `json:"custom_states"`
}

// Opções ajustadas para o Llama 3.1 8B
type OllamaOptions struct {
	Temperature float32 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func saveJSON(dir, filename string, data interface{}) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(dir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func saveMarkdown(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

func generateSpecWithOllama(systemRole, inputDataJSON, targetDir string) {
	fmt.Printf("🦙 Gerando SPEC.md via Ollama (%s) em: %s...\n", OllamaModel, targetDir)

	prompt := fmt.Sprintf(`System: %s
Responda diretamente em Markdown sem introduções ou despedidas. Seja direto, técnico e estruture com títulos claros e listas.

Entrada JSON do Bubble.io:
`+"```json"+`
%s
`+"```", systemRole, inputDataJSON)

	reqPayload := OllamaRequest{
		Model:  OllamaModel,
		Prompt: prompt,
		Stream: false,
		Options: OllamaOptions{
			Temperature: 0.2,   // Baixa temperatura para manter a resposta estritamente analítica e sem alucinações
			NumCtx:      16384, // Janela expandida aproveitando a capacidade nativa do Llama 3.1
		},
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		fmt.Printf("❌ Erro ao serializar requisição Ollama: %v\n", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Post(OllamaEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("⚠️  Não foi possível conectar ao Ollama em %s: %v\n", OllamaEndpoint, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Erro ao ler resposta do Ollama: %v\n", err)
		return
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		fmt.Printf("❌ Erro no parse de retorno do Ollama: %v\n", err)
		return
	}

	if err := saveMarkdown(targetDir, "SPEC.md", ollamaResp.Response); err != nil {
		fmt.Printf("❌ Erro ao salvar SPEC.md: %v\n", err)
	} else {
		fmt.Printf("✅ SPEC.md criado em %s/SPEC.md\n", targetDir)
	}
}

func truncateData(data string, maxChars int) string {
	if len(data) > maxChars {
		return data[:maxChars] + "\n...[dados condensados para análise]..."
	}
	return data
}

func main() {
	inputFileName := "gestao-contratos-conexao.bubble"
	outputBaseDir := "extracted_app"

	if len(os.Args) > 1 {
		inputFileName = os.Args[1]
	}

	fmt.Printf("⚡ Decompilando [%s] com auxílio do %s...\n", inputFileName, OllamaModel)
	start := time.Now()

	file, err := os.Open(inputFileName)
	if err != nil {
		fmt.Printf("❌ Erro ao abrir arquivo: %v\n", err)
		return
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("❌ Erro ao ler arquivo: %v\n", err)
		return
	}

	var app BubbleApp
	if err := json.Unmarshal(byteValue, &app); err != nil {
		fmt.Printf("❌ Erro no parse do JSON do Bubble: %v\n", err)
		return
	}

	// 1. CAMADA DE BANCO DE DADOS
	fmt.Println("\n📦 1/3 - Camada de Banco de Dados...")
	dbDir := filepath.Join(outputBaseDir, "01_database")

	if len(app.OptionSets) > 0 {
		saveJSON(dbDir, "option_sets.json", app.OptionSets)
	} else if len(app.CustomOptions) > 0 {
		saveJSON(dbDir, "option_sets.json", app.CustomOptions)
	}

	reCustomTypes := regexp.MustCompile(`custom\.[a-zA-Z0-9_]+`)
	matchesTypes := reCustomTypes.FindAllString(string(byteValue), -1)
	uniqueTypes := make(map[string]bool)
	for _, t := range matchesTypes {
		uniqueTypes[t] = true
	}
	detectedTypes := make([]string, 0, len(uniqueTypes))
	for t := range uniqueTypes {
		detectedTypes = append(detectedTypes, t)
	}
	saveJSON(dbDir, "detected_custom_tables.json", detectedTypes)

	dbPrompt := "Você é um Engenheiro de Banco de Dados. Analise as tabelas e enums do Bubble.io. Forneça: 1) O dicionário das entidades encontradas; 2) Proposta de modelo relacional normalizado; 3) O script SQL DDL para PostgreSQL/Supabase com chaves estrangeiras, UUIDs e índices."
	sampleDB := fmt.Sprintf("Tabelas detectadas: %v\nOpções/Enums: %s", detectedTypes, string(app.OptionSets))
	generateSpecWithOllama(dbPrompt, truncateData(sampleDB, 12000), dbDir)

	// 2. CAMADA DE WORKFLOWS & 3. CAMADA DE UI
	fmt.Println("\n⚙️  2/3 e 3/3 - Camadas de Workflows e Interface por Página...")
	wfDir := filepath.Join(outputBaseDir, "02_workflows")
	uiDir := filepath.Join(outputBaseDir, "04_ui_pages")

	for key, page := range app.Pages {
		name := page.Name
		if strings.TrimSpace(name) == "" {
			name = key
		}

		pageUIDir := filepath.Join(uiDir, name)
		saveJSON(pageUIDir, "elements_tree.json", page.Elements)
		saveJSON(pageUIDir, "page_meta.json", map[string]interface{}{
			"id":            key,
			"name":          page.Name,
			"properties":    page.Properties,
			"custom_states": page.CustomStates,
		})

		// Workflows da Página
		if len(page.Workflows) > 0 {
			pageWfDir := filepath.Join(wfDir, "pages", name)
			saveJSON(pageWfDir, fmt.Sprintf("%s_workflows.json", name), page.Workflows)

			wfBytes, _ := json.MarshalIndent(page.Workflows, "", "  ")
			wfPrompt := fmt.Sprintf("Você é um Arquiteto de Software Backend. Analise os workflows da tela '%s' do Bubble.io. Documente cada fluxo: evento gatilho, condicionais ('Only when'), ações de banco executadas e como reescrever essa regra em uma Server Action / Service TypeScript desacoplado.", name)
			generateSpecWithOllama(wfPrompt, truncateData(string(wfBytes), 12000), pageWfDir)
		}

		// Telas e Componentes
		elemBytes, _ := json.MarshalIndent(page.Elements, "", "  ")
		uiPrompt := fmt.Sprintf("Você é um Frontend Architect especializado em React e Tailwind CSS. Analise a árvore de componentes da tela '%s' do Bubble.io. Descreva a hierarquia visual (cards, inputs, tabelas/repeating groups), estados dinâmicos e especifique os componentes necessários para recriar esta página.", name)
		generateSpecWithOllama(uiPrompt, truncateData(string(elemBytes), 12000), pageUIDir)
	}

	fmt.Printf("\n✨ Processamento e geração de especificações concluídos em %v!\n", time.Since(start))
	fmt.Printf("📁 Verifique as pastas geradas em ./%s/\n", outputBaseDir)
}