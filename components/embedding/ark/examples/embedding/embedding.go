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

	"github.com/cloudwego/eino-ext/components/embedding/ark"
)

func main() {
	ctx := context.Background()

	embedder, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		// you can get key from https://cloud.bytedance.net/ark/region:ark+cn-beijing/endpoint
		// attention: model must support embedding, for example: doubao-embedding
		APIKey: os.Getenv("ARK_API_KEY"), // for example, "xxxxxx-xxxx-xxxx-xxxx-xxxxxxx"
		Model:  os.Getenv("ARK_MODEL"),   // for example, "ep-20240909094235-xxxx"
	})
	if err != nil {
		log.Fatalf("new embedder error: %v\n", err)
		return
	}

	// string embedding
	embeddings, err := embedder.EmbedStrings(ctx, []string{"hello world", "hello world"})
	if err != nil {
		log.Fatalf("embedding error: %v\n", err)
		return
	}

	log.Printf("string embedding: %v\n", embeddings)

	// multi-modal embedding
	embedding, err := embedder.EmbedContents(ctx, []*schema.ChatMessagePart{
		{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL: "https://ck-test.tos-cn-beijing.volces.com/vlm/pexels-photo-27163466.jpeg",
			},
		},
		{
			Type: schema.ChatMessagePartTypeText,
			Text: "图片分别有什么",
		},
	})
	if err != nil {
		log.Fatalf("embedding error: %v\n", err)
	}
	log.Printf("multi-modal embedding: %+v\n", embedding)
}
