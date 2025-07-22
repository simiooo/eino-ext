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

package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
)

func main() {
	ctx := context.Background()
	// see: https://help.aliyun.com/zh/model-studio/developer-reference/text-embedding-synchronous-api?spm=a2c4g.11186623.help-menu-2400256.d_3_3_9_2.7ad630b71huT6w
	apiKey := os.Getenv("DASHSCOPE_API_KEY")

	// string embedding
	embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		APIKey: apiKey,
		Model:  "text-embedding-v3",
	})
	if err != nil {
		log.Printf("new embedder error: %v\n", err)
		return
	}

	embedding, err := embedder.EmbedStrings(ctx, []string{"hello world", "bye bye"})
	if err != nil {
		log.Printf("embedding error: %v\n", err)
		return
	}

	log.Printf("string embedding: %v\n", embedding)

	// multi-modal embedding
	embedder, err = dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		APIKey: apiKey,
		Model:  "multimodal-embedding-v1",
	})
	if err != nil {
		log.Printf("new embedder error: %v\n", err)
		return
	}

	embedding, err = embedder.EmbedContents(ctx, []*schema.ChatMessagePart{
		{
			Type: schema.ChatMessagePartTypeText,
			Text: "通用多模态表征模型",
		},
		{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL: "https://mitalinlp.oss-cn-hangzhou.aliyuncs.com/dingkun/images/1712648554702.jpg",
			},
		},
		{
			Type: schema.ChatMessagePartTypeVideoURL,
			VideoURL: &schema.ChatMessageVideoURL{
				URL: "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250107/lbcemt/new+video.mp4",
			},
		},
	})
	if err != nil {
		log.Printf("embedding error: %v\n", err)
		return
	}

	log.Printf("multi-modal embedding: %v\n", embedding)
}
