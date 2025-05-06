package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

const (
	SevenDayILMPolicyName = "seven-days-ilm-policy"
)

// ESClient Elasticsearch client structure
type ESClient struct {
	client *elasticsearch.Client
}

// ESConfig Elasticsearch
type ESConfig struct {
	Addresses []string
	Username  string
	Password  string
}

func NewESClient(config ESConfig) (*ESClient, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: config.Addresses,
		Username:  config.Username,
		Password:  config.Password,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ES client")
	}
	return &ESClient{client: client}, nil
}

// es info
func (es *ESClient) Info() (map[string]any, error) {
	res, err := es.client.Info()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ES info")
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, errors.New("failed to get ES info: " + res.String())
	}
	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "failed to parse ES info response")
	}
	return result, nil
}

// CreateIndex creates an index using template with 7-day deletion lifecycle policy
func (es *ESClient) CreateIndex(ctx context.Context, indexName string, mappings map[string]any) error {
	// 创建模板名称
	templateName := fmt.Sprintf("%s-template", indexName)

	// 使用模板创建
	if err := es.CreateIndexTemplate(ctx, templateName, indexName, mappings); err != nil {
		return errors.Wrap(err, "failed to create index template")
	}

	// 创建第一个索引
	firstIndex := es.GetFirstIndexName(indexName)

	exists, err := es.IndexExists(ctx, firstIndex)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to check existence of first index %s", firstIndex))
	}

	if !exists {
		req := esapi.IndicesCreateRequest{
			Index: firstIndex,
		}

		res, createIndexErr := req.Do(ctx, es.client)
		if createIndexErr != nil {
			return errors.Wrap(createIndexErr, "failed to create first index")
		}
		defer res.Body.Close()

		if res.IsError() {
			// Consider specific error handling, e.g., if it's a non-404 error after checking existence
			return errors.New("failed to create first index: " + res.String())
		}
	}

	// 创建别名
	aliasReq := esapi.IndicesUpdateAliasesRequest{
		Body: strings.NewReader(fmt.Sprintf(`{
            "actions": [
                {
                    "add": {
                        "index": "%s",
                        "alias": "%s"
                    }
                },
                {
                    "add": {
                        "index": "%s",
                        "alias": "%s",
                        "is_write_index": true
                    }
                }
            ]
        }`, firstIndex, indexName, firstIndex, es.GetIndexWrite(indexName))),
	}

	aliasRes, err := aliasReq.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to create aliases")
	}
	defer aliasRes.Body.Close()

	if aliasRes.IsError() {
		return errors.New("failed to create aliases: " + aliasRes.String())
	}

	return nil
}

// IndexDocument indexes a document
func (es *ESClient) IndexDocument(ctx context.Context, indexName string, document any) error {
	documentBytes, err := json.Marshal(document)
	if err != nil {
		return errors.Wrap(err, "failed to serialize document")
	}

	req := esapi.IndexRequest{
		Index:   indexName,
		Body:    bytes.NewReader(documentBytes),
		Refresh: "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to index document request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to index document: " + res.String())
	}
	return nil
}

// SearchByKeyword performs fuzzy search map[string]string field keyword with pagination
func (es *ESClient) SearchByKeyword(ctx context.Context, indexName string, fieldKeyword map[string][]string, page, size int) ([]map[string]any, error) {
	from := (page - 1) * size
	from = max(0, from)
	size = max(1, size)

	query := map[string]any{
		"query": map[string]any{
			"terms": fieldKeyword,
		},
		"from": from,
		"size": size,
		"sort": []map[string]any{
			{
				"@timestamp": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	return es.search(ctx, indexName, query)
}

func (es *ESClient) SearchByMatch(ctx context.Context, indexName string, matchMap map[string]string, page, size int) ([]map[string]any, error) {
	from := (page - 1) * size
	from = max(0, from)
	size = max(1, size)

	query := map[string]any{
		"query": map[string]any{
			"match": matchMap,
		},
		"from": from,
		"size": size,
		"sort": []map[string]any{
			{
				"@timestamp": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	return es.search(ctx, indexName, query)
}

func (es *ESClient) SearchByKeywordAndMatch(ctx context.Context, indexName string, keywordMap map[string][]string, matchMap map[string]string, page, size int) ([]map[string]any, error) {
	from := (page - 1) * size
	from = max(0, from)
	size = max(1, size)

	mustClauses := make([]map[string]any, 0)

	for field, values := range keywordMap {
		if len(values) == 0 {
			mustClauses = append(mustClauses, map[string]any{
				"terms": map[string]any{
					field: values,
				},
			})
		}
	}

	for field, value := range matchMap {
		if value != "" {
			mustClauses = append(mustClauses, map[string]any{
				"match": map[string]any{
					field: value,
				},
			})
		}
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": mustClauses,
			},
		},
		"from": from,
		"size": size,
		"sort": []map[string]any{
			{
				"@timestamp": map[string]any{
					"order": "desc",
				},
			},
		},
	}

	return es.search(ctx, indexName, query)
}

// search internal search method
func (es *ESClient) search(ctx context.Context, indexName string, query map[string]any) ([]map[string]any, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, err
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

// IndexExists checks if an index exists
func (es *ESClient) IndexExists(ctx context.Context, indexName string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, es.client) // Changed to use passed ctx
	if err != nil {
		return false, errors.Wrap(err, "failed to check index existence")
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return false, nil
	}

	if res.StatusCode == 200 {
		return true, nil
	}

	return false, errors.New("failed to check index existence: " + res.String())
}

// ILMPolicyExists checks if an ILM policy exists
func (es *ESClient) ILMPolicyExists(ctx context.Context, policyName string) (bool, error) {
	req := esapi.ILMGetLifecycleRequest{
		Policy: policyName,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return false, errors.Wrap(err, "failed to check ILM policy existence")
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return false, nil
	}

	if res.StatusCode == 200 {
		return true, nil
	}

	return false, errors.New("failed to check ILM policy existence: " + res.String())
}

// CreateSevenDaysILMPolicy creates an ILM policy that deletes indices after 7 days
func (es *ESClient) CreateSevenDaysILMPolicy(ctx context.Context, policyName string) error {
	// 首先检查策略是否已存在
	exists, err := es.ILMPolicyExists(ctx, policyName)
	if err != nil {
		return errors.Wrap(err, "failed to check ILM policy existence")
	}

	if exists {
		return nil // 策略已存在，无需重复创建
	}

	policy := map[string]any{
		"policy": map[string]any{
			"phases": map[string]any{
				"hot": map[string]any{
					"min_age": "0ms",
					"actions": map[string]any{
						"rollover": map[string]any{
							"max_age":  "1d",
							"max_size": "5gb",
						},
					},
				},
				"delete": map[string]any{
					"min_age": "7d",
					"actions": map[string]any{
						"delete": map[string]any{},
					},
				},
			},
		},
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ILM policy")
	}

	req := esapi.ILMPutLifecycleRequest{
		Policy: policyName,
		Body:   bytes.NewReader(policyBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to create ILM policy request")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to create ILM policy: " + res.String())
	}

	return nil
}

// CreateIndexTemplate creates an index template with 7-day deletion lifecycle policy
func (es *ESClient) CreateIndexTemplate(ctx context.Context, templateName, indexName string, mappings map[string]any) error {
	req := esapi.IndicesGetIndexTemplateRequest{
		Name: templateName,
	}

	res, err := req.Do(ctx, es.client)
	if err == nil && !res.IsError() {
		return nil
	}
	// If err is not nil OR res.IsError(), proceed to create.
	// This logic is fine for checking existence.

	if err = es.CreateSevenDaysILMPolicy(ctx, SevenDayILMPolicyName); err != nil {
		return errors.Wrap(err, "failed to create ILM policy")
	}

	settings := map[string]any{
		"index": map[string]any{
			"lifecycle": map[string]any{
				"name":           SevenDayILMPolicyName,
				"rollover_alias": es.GetIndexWrite(indexName),
			},
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	}

	template := map[string]any{
		"index_patterns": []string{es.GetIndexPatternName(indexName)},
		"template": map[string]any{
			"settings": settings,
			"mappings": mappings,
		},
	}

	templateBytes, err := json.Marshal(template)
	if err != nil {
		return errors.Wrap(err, "failed to marshal template")
	}

	putReq := esapi.IndicesPutIndexTemplateRequest{
		Name: templateName,
		Body: bytes.NewReader(templateBytes),
	}

	putRes, err := putReq.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to create index template request")
	}
	defer putRes.Body.Close()

	if putRes.IsError() {
		return errors.New("failed to create index template: " + putRes.String())
	}

	return nil
}

func (es *ESClient) GetIndexWrite(prefix string) string {
	return fmt.Sprintf("%s-write", prefix)
}

func (es *ESClient) GetIndexPatternName(prefix string) string {
	return fmt.Sprintf("%s-*", prefix)
}

func (es *ESClient) GetFirstIndexName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, time.Now().Format("2006.01.02"))
}
