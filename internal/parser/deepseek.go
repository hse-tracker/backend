package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
)

type LLMResponse struct {
	StudentNameColumnIndex int            `json:"student_name_column_index"`
	DataStartRow           int            `json:"data_start_row"`
	Structure              []LLMGradeNode `json:"structure"`
}

type LLMGradeNode struct {
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	ColumnIndex    *int           `json:"column_index"`
	Weight         *float64       `json:"weight"`
	DisplayFormula *string        `json:"display_formula"`
	Children       []LLMGradeNode `json:"children"`
}

// sends ALL subject info to deepseek
// depth 3
func AnalyzeTableWithDeepSeek(ctx context.Context, apiKey, tableContext string) (*LLMResponse, error) {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.deepseek.com/v1"
	client := openai.NewClientWithConfig(config)

	systemPrompt :=
		`Ты - эксперт по анализу образовательных ведомостей из Google Sheets.
Твоя задача: изучить предоставленные данные (первые несколько строк, включая формулы) и вернуть СТРОГИЙ JSON со структурой оценок.

ПРАВИЛА:
1. Индексы: Найди student_name_column_index (0 для A, 1 для B) и data_start_row (строка начала данных студентов, счет с 0).
2. ИГНОРИРОВАНИЕ ОКРУГЛЕНИЙ (КРИТИЧЕСКИ ВАЖНО): Часто в таблицах есть колонка с точным баллом (например, "Итог", "Накоп", 8.50) и колонка с округлением (например, "Округл", "Округление", 9.00). Ты ОБЯЗАН полностью игнорировать колонку с округлением! Главным (корневым) узлом структуры должна быть оценка БЕЗ округления (точный итог). Вообще не включай колонку с округлением в итоговый JSON.
3. Дерево: Восстанови иерархию: type: "value" - конечная оценка, например "ДЗ1", "КР1", "Экзамен"; type: "formula" - ФИНАЛЬНАЯ ИТОГОВАЯ ОЦЕНКА БЕЗ ОКРУГЛЕНИЯ, во всей таблице должна быть ТОЛЬКО ОДНА; type: "folder" - оценка, которая включает в себя дочерние оценки, например "ДЗ avg".
4. ВИРТУАЛЬНЫЕ ПАПКИ: Если у оценки с типом "folder" нет собственной колонки, и она агрегирует колонки (например СРЕДНЕЕ(C,D)) только в родительской формуле, то задай ей "column_index: null" и помести C и D внутрь как детей. Если у оценки с типом "folder" есть собственная колонка, то укажи ее в column_index.
5. ВЕСА (weight): Каждый элемент с типом "folder" имеет вес в итоговой оценке, укажи его в виде float (например, 0.4 для 40%). Вес папки - это суммарный вес ее детей в родительской формуле.
6. ФОРМУЛЫ (display_formula): Значение display_formula должно быть только у элемента с типом formula (то есть у финальной итоговой оценки). display_formula - красивая, понятная человеку строка расчета (например "0.4 * Домашние задания + 0.6 * КР1"), для всех остальных типов укажи null.

Пример ответа:

{
  "student_name_column_index": 0,
  "data_start_row": 2,
  "structure":[
    {
      "name": "Итог",
      "type": "formula",
      "column_index": 5,
      "weight": null,
      "display_formula": "0.4 * Домашние задания + 0.6 * КР1",
      "children":[
        {
          "name": "Домашние задания",
          "type": "folder",
          "column_index": null,
          "weight": 0.4,
          "display_formula": null,
          "children":[
            { "name": "ДЗ1", "type": "value", "column_index": 2, "weight": null, "display_formula": null, "children": [] },
            { "name": "ДЗ2", "type": "value", "column_index": 3, "weight": null, "display_formula": null, "children": [] }
          ]
        },
        {
          "name": "КР1",
          "type": "value",
          "column_index": 4,
          "weight": 0.6,
          "display_formula": null,
          "children": []
        }
      ]
    }
  ]
}`

	userPrompt := fmt.Sprintf("Проанализируй таблицу и верни структуру (тип узлов: folder, formula, value).\n\nТаблица:\n%s", tableContext)

	req := openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.1,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("deepseek api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from DeepSeek")
	}

	rawJSON := resp.Choices[0].Message.Content

	// debug log
	log.Println("- - - AnalyzeTableWithDeepSeek - - -")
	log.Println(rawJSON)

	var result LLMResponse
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from LLM: %w\nRaw: %s", err, rawJSON)
	}

	return &result, nil
}
