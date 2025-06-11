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

// IndexDocument indexes a document
func (es *ESClient) IndexDocument(ctx context.Context, indexName string, document map[string]any) error {
	if _, exists := document["@timestamp"]; !exists {
		document["@timestamp"] = time.Now().UTC()
	}
	documentBytes, err := json.Marshal(document)
	if err != nil {
		return errors.Wrap(err, "failed to serialize document")
	}

	req := esapi.IndexRequest{
		Index:   es.GetIndexWrite(indexName),
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

// BulkIndex performs bulk indexing of documents
func (es *ESClient) BulkIndex(ctx context.Context, indexName string, documents []map[string]any) error {
	var buf bytes.Buffer

	indexName = es.GetIndexWrite(indexName)
	for _, doc := range documents {
		if _, exists := doc["@timestamp"]; !exists {
			doc["@timestamp"] = time.Now().UTC()
		}
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

	res, err := req.Do(ctx, es.client)
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

	if val, ok := result["errors"]; ok && val.(bool) {
		return errors.New("some documents failed to index")
	}

	return nil
}

// CreateIndex creates an index using template with 7-day deletion lifecycle policy
func (es *ESClient) CreateIndex(ctx context.Context, indexName string, mappings map[string]any) error {
	templateName := fmt.Sprintf("%s-template", indexName)

	if err := es.CreateIndexTemplate(ctx, templateName, indexName, mappings); err != nil {
		return errors.Wrap(err, "failed to create index template")
	}

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

	aliasExistsReq := esapi.IndicesGetAliasRequest{
		Name: []string{indexName, es.GetIndexWrite(indexName)},
	}

	aliasExistsRes, err := aliasExistsReq.Do(ctx, es.client)
	if err == nil && !aliasExistsRes.IsError() {
		return nil
	}

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
		return err
	}
	defer aliasRes.Body.Close()

	if aliasRes.IsError() {
		return errors.New("failed to create aliases: " + aliasRes.String())
	}

	return nil
}

// SearchByKeyword performs fuzzy search map[string]string field keyword with pagination
func (es *ESClient) SearchByKeyword(ctx context.Context, indexName string, fieldKeyword map[string][]string, page, size int) (SearchResult, error) {
	from := (page - 1) * size
	from = max(0, from)
	size = max(1, size)

	shouldClauses := make([]map[string]any, 0)
	for field, values := range fieldKeyword {
		if len(values) > 0 {
			shouldClauses = append(shouldClauses, map[string]any{
				"terms": map[string]any{
					field: values,
				},
			})
		}
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"should": shouldClauses,
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

func (es *ESClient) SearchByMatch(ctx context.Context, indexName string, matchMap map[string]string, page, size int) (SearchResult, error) {
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

func (es *ESClient) SearchByKeywordAndMatch(ctx context.Context, indexName string, keywordMap map[string][]string, matchMap map[string]string, page, size int) (SearchResult, error) {
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

type SearchResponse struct {
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64     `json:"max_score"`
		Hits     []SearchHit `json:"hits"`
	} `json:"hits"`
	TimedOut bool   `json:"timed_out"`
	Took     int    `json:"took"`
	Shards   Shards `json:"_shards"`
}

type SearchHit struct {
	Index  string          `json:"_index"`
	ID     string          `json:"_id"`
	Score  float64         `json:"_score"`
	Source json.RawMessage `json:"_source"`
}

type Shards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type SearchResult struct {
	Total int              `json:"total"`
	Data  []map[string]any `json:"data"`
}

func (es *ESClient) search(ctx context.Context, indexName string, query map[string]any) (SearchResult, error) {
	searchResult := SearchResult{}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return searchResult, err
	}

	req := esapi.SearchRequest{
		Index: []string{es.GetIndexPatternName(indexName)},
		Body:  bytes.NewReader(queryBytes),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return searchResult, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return searchResult, errors.New("search failed: " + res.String())
	}

	var searchResponse SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return searchResult, errors.Wrap(err, "failed to parse search results")
	}

	documents := make([]map[string]any, len(searchResponse.Hits.Hits))
	for i, hit := range searchResponse.Hits.Hits {
		var doc map[string]any
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			return searchResult, errors.Wrap(err, "failed to parse document source")
		}
		documents[i] = doc
	}
	searchResult.Data = documents
	searchResult.Total = searchResponse.Hits.Total.Value
	return searchResult, nil
}

// DeleteIndex deletes an index
func (es *ESClient) DeleteIndex(ctx context.Context, indexName string) error {
	req := esapi.IndicesGetRequest{
		Index: []string{es.GetIndexPatternName(indexName)},
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to get indices")
	}
	defer res.Body.Close()

	if res.IsError() {
		return errors.New("failed to get indices: " + res.String())
	}

	var indices map[string]any
	if err = json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return errors.Wrap(err, "failed to parse indices response")
	}

	var actualIndices []string
	for index := range indices {
		actualIndices = append(actualIndices, index)
	}

	if len(actualIndices) == 0 {
		return nil
	}

	deleteReq := esapi.IndicesDeleteRequest{
		Index: actualIndices,
	}

	deleteRes, err := deleteReq.Do(ctx, es.client)
	if err != nil {
		return errors.Wrap(err, "failed to delete indices")
	}
	defer deleteRes.Body.Close()

	if deleteRes.IsError() {
		return errors.New("failed to delete indices: " + deleteRes.String())
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
	exists, err := es.ILMPolicyExists(ctx, policyName)
	if err != nil {
		return errors.Wrap(err, "failed to check ILM policy existence")
	}

	if exists {
		return nil
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
