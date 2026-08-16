package memory

import (
	"net/url"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

func NewQdrantStore(
	collectionName string,
	embedderClient embeddings.EmbedderClient,
	qdrantURL string,
) (vectorstores.VectorStore, error) {
	embedder, err := embeddings.NewEmbedder(embedderClient)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(qdrantURL)
	if err != nil {
		return nil, err
	}
	store, err := qdrant.New(
		qdrant.WithURL(*url),
		qdrant.WithCollectionName(collectionName),
		qdrant.WithEmbedder(embedder),
	)
	return store, err
}
