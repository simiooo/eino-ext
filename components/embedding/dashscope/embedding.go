/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
)

const (
	baseUrl    = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	dimensions = 1024
)

type EmbeddingConfig struct {
	// APIKey is typically OPENAI_API_KEY, but if you have set up Azure, then it is Azure API_KEY.
	APIKey string `json:"api_key"`
	// Timeout specifies the http request timeout.
	// If HTTPClient is set, Timeout will not be used.
	Timeout time.Duration `json:"timeout"`
	// HTTPClient specifies the client to send HTTP requests.
	// If HTTPClient is set, Timeout will not be used.
	// Optional. Default &http.Client{Timeout: Timeout}
	HTTPClient *http.Client `json:"http_client"`

	// The following fields have the same meaning as the fields in the openai embedding API request.
	// OpenAI Ref: https://platform.openai.com/docs/api-reference/embeddings/create
	// DashScope Ref: https://help.aliyun.com/zh/model-studio/developer-reference/text-embedding-synchronous-api?spm=a2c4g.11186623.help-menu-2400256.d_3_3_9_2.532bf440ali5Ry&scm=20140722.H_2712515._.OR_help-T_cn~zh-V_1

	// Model available models: text_embedding_v / text_embedding_v2 / text_embedding_v3
	// Async embedding models not support.
	Model string `json:"model"`
	// Dimensions specify output vector dimension.
	// Only applicable to text-embedding-v3 model, can only be selected between three values: 1024, 768, and 512.
	// The default value is 1024.
	Dimensions *int `json:"dimensions,omitempty"`
	// Specifies the workspace to be used for this call.
	// Note:
	// - For sub-account API keys, this parameter is REQUIRED (sub-accounts must belong to a workspace).
	// - For main account API keys, this parameter is OPTIONAL.
	Workspace string `json:"workspace"`
}
type Embedder struct {
	cli *openai.EmbeddingClient

	model     string
	httpCli   *http.Client
	apiKey    string
	workspace string
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	encodingFmt := openai.EmbeddingEncodingFormatFloat // only support float currently

	var httpClient *http.Client

	if config.HTTPClient != nil {
		httpClient = config.HTTPClient
	} else {
		httpClient = &http.Client{Timeout: config.Timeout}
	}

	ecfg := &openai.EmbeddingConfig{
		BaseURL:        baseUrl,
		APIKey:         config.APIKey,
		HTTPClient:     httpClient,
		Model:          config.Model,
		EncodingFormat: &encodingFmt,
		Dimensions:     config.Dimensions,
	}

	if ecfg.Dimensions == nil {
		dim := dimensions
		ecfg.Dimensions = &dim
	}

	cli, err := openai.NewEmbeddingClient(ctx, ecfg)
	if err != nil {
		return nil, err
	}

	return &Embedder{
		cli:       cli,
		model:     config.Model,
		httpCli:   httpClient,
		apiKey:    config.APIKey,
		workspace: config.Workspace,
	}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	return e.cli.EmbedStrings(ctx, texts, opts...)
}

func (e *Embedder) EmbedContents(ctx context.Context, contents []*schema.ChatMessagePart, opts ...embedding.Option) (result [][]float64, err error) {
	o := embedding.GetCommonOptions(&embedding.Options{Model: &e.model}, opts...)
	cbConf := &embedding.Config{Model: *o.Model}
	ctx = callbacks.OnStart(ctx, &embedding.CallbackInput{
		//Contents: contents, todo
		Config: cbConf,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()
	req := multiModalRequestBody{
		Model: *o.Model,
		Input: &multiModalRequestInput{Contents: make([]map[string]string, 0, len(contents))},
	}
	for _, content := range contents {
		input, err := multiModalContent2Map(content)
		if err != nil {
			return nil, err
		}
		req.Input.Contents = append(req.Input.Contents, input)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(authKey, fmt.Sprintf(bearerFormat, e.apiKey))
	if len(e.workspace) > 0 {
		httpReq.Header.Set(workspaceKey, e.workspace)
	}

	resp, err := e.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to embed multimodal response, status code: %s, body: %s", resp.Status, string(respBody))
	}

	multiModalResp := &multiModalResponseBody{}
	if err := json.Unmarshal(respBody, multiModalResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multimodal response body: %w", err)
	}

	result = make([][]float64, 0, len(multiModalResp.Output.Embeddings))
	for _, emb := range multiModalResp.Output.Embeddings {
		result = append(result, emb.Embedding)
	}

	var tu *embedding.TokenUsage
	if multiModalResp.Usage != nil {
		tu = &embedding.TokenUsage{
			TotalTokens: multiModalResp.Usage.ImageTokens + multiModalResp.Usage.InputTokens,
		}
	}
	callbacks.OnEnd(ctx, &embedding.CallbackOutput{
		Embeddings: result,
		Config:     cbConf,
		TokenUsage: tu,
	})

	return result, nil
}

func multiModalContent2Map(content *schema.ChatMessagePart) (map[string]string, error) {
	if content == nil {
		return nil, fmt.Errorf("content is nil")
	}
	switch content.Type {
	case schema.ChatMessagePartTypeText:
		return map[string]string{
			textType: content.Text,
		}, nil
	case schema.ChatMessagePartTypeImageURL:
		if content.ImageURL == nil {
			return nil, fmt.Errorf("content image_url is nil")
		}
		return map[string]string{
			imageType: content.ImageURL.URL,
		}, nil
	case schema.ChatMessagePartTypeVideoURL:
		if content.VideoURL == nil {
			return nil, fmt.Errorf("content video_url is nil")
		}
		return map[string]string{
			videoType: content.VideoURL.URL,
		}, nil
	default:
		return nil, fmt.Errorf("unknown content type: %v", content.Type)
	}
}

const (
	authKey      = "Authorization"
	bearerFormat = "Bearer %s"
	workspaceKey = "X-DashScope-WorkSpace"

	textType  = "text"
	imageType = "image"
	videoType = "video"
)

type multiModalRequestBody struct {
	Model string                  `json:"model"`
	Input *multiModalRequestInput `json:"input"`
}

type multiModalRequestInput struct {
	Contents []map[string]string `json:"contents"`
}

type multiModalResponseBody struct {
	Output    *multiModalResponseOutput `json:"output"`
	Usage     *multiModalResponseUsage  `json:"usage"`
	RequestID string                    `json:"request_id"`
}

type multiModalResponseOutput struct {
	Embeddings []*multiModalResponseEmbedding `json:"embeddings"`
}

type multiModalResponseEmbedding struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
	Type      string    `json:"type"`
}

type multiModalResponseUsage struct {
	InputTokens int `json:"input_tokens"`
	ImageTokens int `json:"image_tokens"`
}

const typ = "DashScope"

func (e *Embedder) GetType() string {
	return typ
}

func (e *Embedder) IsCallbacksEnabled() bool {
	return true
}
