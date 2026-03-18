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

	req := openai.ChatCompletionRequest{
		Model: "deepseek-chat",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: GoogleSheetsPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: getTablePrompt(tableContext),
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
