package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"
	"github.com/pkg/errors"
)

/*
	// Create client
	config := utils.ESConfig{
	Addresses: []string{"http://localhost:9200"},
	}
	client, err := utils.NewESClient(config)

	// Create index
	mapping := `{
	"mappings": {
		"properties": {
			"title": { "type": "text" },
			"tags": { "type": "keyword" }
		}
	}
	}`
	err = client.CreateIndex("my-index", mapping)

	// Index document
	doc := map[string]any{
	"title": "Test Document",
	"tags":  []string{"test", "demo"},
	}
	err = client.IndexDocument("my-index", "doc1", doc)

	// Search documents
	results, err := client.SearchByTags("my-index", []string{"test"})
*/

// ESClient Elasticsearch client structure
type ESClient struct {
	client *elasticsearch.Client
}

// ESConfig Elasticsearch configuration structure
type ESConfig struct {
	Addresses    []string // ES node address list
	ServiceToken string   // Service Account Token（use K8s env）
	APIKey       string   // API Key
}

// NewESClient creates a new ES client
func NewESClient(config ESConfig) (*ESClient, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:    config.Addresses,
		ServiceToken: config.ServiceToken,
		APIKey:       config.APIKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ES client")
	}
	return &ESClient{client: client}, nil
}

// CreateIndex creates an index
func (es *ESClient) CreateIndex(indexName string, mapping string) error {
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return errors.Wrap(err, "failed to create index request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to create index: " + res.String())
	}

	return nil
}

// IndexDocument indexes a document
func (es *ESClient) IndexDocument(indexName string, documentID string, document any) error {
	documentBytes, err := json.Marshal(document)
	if err != nil {
		return errors.Wrap(err, "failed to serialize document")
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(documentBytes),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return errors.Wrap(err, "failed to index document request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to index document: " + res.String())
	}

	return nil
}

// SearchByTags searches by tags
func (es *ESClient) SearchByTags(indexName string, tags []string) ([]map[string]any, error) {
	query := map[string]any{
		"query": map[string]any{
			"terms": map[string]any{
				"tags": tags,
			},
		},
	}

	return es.search(indexName, query)
}

// SearchByKeyword performs fuzzy search
func (es *ESClient) SearchByKeyword(indexName string, field string, keyword string) ([]map[string]any, error) {
	query := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				field: keyword,
			},
		},
	}

	return es.search(indexName, query)
}

// SearchByDSL performs search using Query DSL
func (es *ESClient) SearchByDSL(indexName string, dsl map[string]any) ([]map[string]any, error) {
	return es.search(indexName, dsl)
}

// search internal search method
func (es *ESClient) search(indexName string, query map[string]any) ([]map[string]any, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize query")
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryBytes),
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute search request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, errors.New("search failed: " + res.String())
	}

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to parse search results")
	}

	hits, found := result["hits"].(map[string]any)["hits"].([]any)
	if !found {
		return make([]map[string]any, 0), nil
	}

	documents := make([]map[string]any, len(hits))
	for i, hit := range hits {
		hitMap := hit.(map[string]any)
		documents[i] = hitMap["_source"].(map[string]any)
	}

	return documents, nil
}

// DeleteIndex deletes an index
func (es *ESClient) DeleteIndex(indexName string) error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return errors.Wrap(err, "failed to delete index request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to delete index: " + res.String())
	}

	return nil
}

// BulkIndex performs bulk indexing of documents
func (es *ESClient) BulkIndex(indexName string, documents []map[string]any) error {
	var buf bytes.Buffer

	for _, doc := range documents {
		meta := map[string]any{
			"index": map[string]any{
				"_index": indexName,
			},
		}

		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			return errors.Wrap(err, "failed to encode metadata")
		}

		if err := json.NewEncoder(&buf).Encode(doc); err != nil {
			return errors.Wrap(err, "failed to encode document")
		}
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return errors.Wrap(err, "failed to execute bulk index request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("bulk indexing failed: " + res.String())
	}

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return errors.Wrap(err, "failed to parse bulk index results")
	}

	if result["errors"].(bool) {
		return errors.New("some documents failed to index")
	}

	return nil
}
