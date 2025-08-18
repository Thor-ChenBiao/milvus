// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/v2/milvuspb"
	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/milvus-io/milvus/internal/types"
	"github.com/milvus-io/milvus/pkg/v2/util"
	"github.com/milvus-io/milvus/pkg/v2/util/merr"
	"google.golang.org/protobuf/proto"
)

// ToolsCatalog manages all available MCP tools
type ToolsCatalog struct {
	proxy types.ProxyComponent
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewToolsCatalog creates a new tools catalog
func NewToolsCatalog(proxy types.ProxyComponent) *ToolsCatalog {
	tc := &ToolsCatalog{
		proxy: proxy,
		tools: make(map[string]*Tool),
	}
	tc.registerAll()
	return tc
}

// Get retrieves a tool by name
func (tc *ToolsCatalog) Get(name string) (*Tool, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	tool, exists := tc.tools[name]
	return tool, exists
}

// List returns all tools as descriptions
func (tc *ToolsCatalog) List() []McpToolDescription {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	descriptions := make([]McpToolDescription, 0, len(tc.tools))
	for _, tool := range tc.tools {
		descriptions = append(descriptions, McpToolDescription{
			Name:         tool.Name,
			Title:        tool.Title,
			Description:  tool.Description,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
		})
	}
	return descriptions
}

// register adds a tool to the catalog
func (tc *ToolsCatalog) register(tool *Tool) {
	tc.tools[tool.Name] = tool
}

// registerAll registers all available tools
func (tc *ToolsCatalog) registerAll() {
	// Collection management tools
	tc.register(&Tool{
		Name:         ToolCollectionListName,
		Title:        ToolCollectionListTitle,
		Description:  ToolCollectionListDescription,
		Execute:      tc.listCollections,
		InputSchema:  tc.schemaForListCollections(),
		OutputSchema: tc.outputSchemaForListCollections(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeDatabase, ObjectPrivilege: PrivShowCollections, ObjectNameField: ParamDatabaseKey},
		},
	})

	tc.register(&Tool{
		Name:         ToolCollectionCreateName,
		Title:        ToolCollectionCreateTitle,
		Description:  ToolCollectionCreateDescription,
		Execute:      tc.createCollection,
		InputSchema:  tc.schemaForCreateCollection(),
		OutputSchema: tc.outputSchemaForCreateCollection(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeDatabase, ObjectPrivilege: PrivCreateCollection, ObjectNameField: ParamDatabaseKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolCollectionDescribeName,
		Title:       ToolCollectionDescribeTitle,
		Description: ToolCollectionDescribeDescription,
		Execute:     tc.describeCollection,
		InputSchema: tc.schemaForDescribeCollection(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivDescribeCollection, ObjectNameField: ParamCollectionNameKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolCollectionDropName,
		Title:       ToolCollectionDropTitle,
		Description: ToolCollectionDropDescription,
		Execute:     tc.dropCollection,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivDropCollection, ObjectNameField: ParamCollectionNameKey},
		},
	})

	// Data operation tools
	tc.register(&Tool{
		Name:        ToolDataInsertName,
		Title:       ToolDataInsertTitle,
		Description: ToolDataInsertDescription,
		Execute:     tc.insertData,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivInsert, ObjectNameField: ParamCollectionNameKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolDataSearchName,
		Title:       ToolDataSearchTitle,
		Description: ToolDataSearchDescription,
		Execute:     tc.searchVectors,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivSearch, ObjectNameField: ParamCollectionNameKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolDataQueryName,
		Title:       ToolDataQueryTitle,
		Description: ToolDataQueryDescription,
		Execute:     tc.queryData,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivQuery, ObjectNameField: ParamCollectionNameKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolDataDeleteName,
		Title:       ToolDataDeleteTitle,
		Description: ToolDataDeleteDescription,
		Execute:     tc.deleteData,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivDelete, ObjectNameField: ParamCollectionNameKey},
		},
	})

	// Index management tools
	tc.register(&Tool{
		Name:        ToolIndexCreateName,
		Title:       ToolIndexCreateTitle,
		Description: ToolIndexCreateDescription,
		Execute:     tc.createIndex,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivCreateIndex, ObjectNameField: ParamCollectionNameKey},
		},
	})

	tc.register(&Tool{
		Name:        ToolIndexDescribeName,
		Title:       ToolIndexDescribeTitle,
		Description: ToolIndexDescribeDescription,
		Execute:     tc.describeIndex,
		InputSchema: tc.simpleCollectionSchema(),
		RequiredPrivileges: []PrivilegeRequirement{
			{ObjectType: ObjectTypeCollection, ObjectPrivilege: PrivDescribeIndex, ObjectNameField: ParamCollectionNameKey},
		},
	})
}

// Tool implementations

func (tc *ToolsCatalog) listCollections(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	dbName := args.GetString("database", "default")

	req := &milvuspb.ShowCollectionsRequest{
		DbName: dbName,
	}
	resp, err := tc.proxy.ShowCollections(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := merr.Error(resp.GetStatus()); err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"collections": resp.CollectionNames,
		"database":    dbName,
	}

	message := fmt.Sprintf(MsgCollectionListSuccess, len(resp.CollectionNames), dbName)
	return NewToolResultWithData(message, data), nil
}

func (tc *ToolsCatalog) createCollection(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Validate required parameters
	if err := args.Require(ParamCollectionNameKey, ParamDimensionKey); err != nil {
		return nil, err
	}

	dbName := args.GetString(ParamDatabaseKey, util.DefaultDBName)
	collectionName := args.GetString(ParamCollectionNameKey, "")
	dimension := args.GetInt(ParamDimensionKey, 0)
	metricType := args.GetString(ParamMetricTypeKey, DefaultMetricType)

	if dimension <= 0 {
		return nil, fmt.Errorf(ErrDimensionPositive, dimension)
	}

	// Create simple schema with auto ID
	schema := &schemapb.CollectionSchema{
		Name:        collectionName,
		Description: DefaultCollectionDescription,
		Fields: []*schemapb.FieldSchema{
			{
				FieldID:      100,
				Name:         FieldPrimaryIDName,
				IsPrimaryKey: true,
				DataType:     schemapb.DataType_Int64,
				AutoID:       true,
			},
			{
				FieldID:  101,
				Name:     FieldVectorName,
				DataType: schemapb.DataType_FloatVector,
				TypeParams: []*commonpb.KeyValuePair{
					{Key: TypeParamDimKey, Value: fmt.Sprintf("%d", dimension)},
				},
			},
		},
	}

	// Use protobuf wire format as required by Milvus
	schemaBytes, err := proto.Marshal(schema)
	if err != nil {
		return nil, err
	}

	req := &milvuspb.CreateCollectionRequest{
		DbName:         dbName,
		CollectionName: collectionName,
		Schema:         schemaBytes,
	}

	resp, err := tc.proxy.CreateCollection(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.GetCode() != 0 {
		return nil, fmt.Errorf(resp.GetReason())
	}

	// Auto-create index for better performance
	indexReq := &milvuspb.CreateIndexRequest{
		DbName:         dbName,
		CollectionName: collectionName,
		FieldName:      FieldVectorName,
		IndexName:      DefaultIndexName,
		ExtraParams: []*commonpb.KeyValuePair{
			{Key: IndexParamIndexTypeKey, Value: DefaultIndexType},
			{Key: IndexParamMetricTypeKey, Value: metricType},
			{Key: IndexParamParamsKey, Value: `{"nlist": 128}`},
		},
	}

	tc.proxy.CreateIndex(ctx, indexReq)

	data := map[string]interface{}{
		"collection_name": collectionName,
		"database":        dbName,
		"dimension":       dimension,
		"metric_type":     metricType,
		"status":          "created",
	}

	message := fmt.Sprintf(MsgCollectionCreateSuccess, collectionName, dimension)
	return NewToolResultWithData(message, data), nil
}

func (tc *ToolsCatalog) describeCollection(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	if err := args.Require(ParamCollectionNameKey); err != nil {
		return nil, err
	}

	dbName := args.GetString(ParamDatabaseKey, util.DefaultDBName)
	collectionName := args.GetString(ParamCollectionNameKey, "")

	resp, err := tc.proxy.DescribeCollection(ctx, &milvuspb.DescribeCollectionRequest{
		DbName:         dbName,
		CollectionName: collectionName,
	})
	if err != nil {
		return nil, err
	}

	if err := merr.Error(resp.GetStatus()); err != nil {
		return nil, err
	}

	schema := resp.Schema

	fields := make([]map[string]interface{}, 0)
	for _, field := range schema.Fields {
		fieldInfo := map[string]interface{}{
			"name":       field.Name,
			"type":       field.DataType.String(),
			"is_primary": field.IsPrimaryKey,
			"auto_id":    field.AutoID,
		}

		// Add dimension for vector fields
		for _, param := range field.TypeParams {
			if param.Key == TypeParamDimKey {
				fieldInfo["dimension"] = param.Value
			}
		}

		fields = append(fields, fieldInfo)
	}

	data := map[string]interface{}{
		"collection_name":   collectionName,
		"database":          dbName,
		"created_time":      resp.CreatedTimestamp,
		"fields":            fields,
		"consistency_level": resp.ConsistencyLevel.String(),
	}

	message := fmt.Sprintf(MsgCollectionDescribeSuccess, collectionName, len(fields))
	return NewToolResultWithData(message, data), nil
}

func (tc *ToolsCatalog) dropCollection(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	if err := args.Require("collection_name"); err != nil {
		return nil, err
	}

	dbName := args.GetString("database", "default")
	collectionName := args.GetString("collection_name", "")

	resp, err := tc.proxy.DropCollection(ctx, &milvuspb.DropCollectionRequest{
		DbName:         dbName,
		CollectionName: collectionName,
	})
	if err != nil {
		return nil, err
	}

	if resp.GetCode() != 0 {
		return nil, fmt.Errorf(resp.GetReason())
	}

	data := map[string]interface{}{
		"collection_name": collectionName,
		"database":        dbName,
		"status":          "dropped",
	}

	message := fmt.Sprintf(MsgCollectionDropSuccess, collectionName)
	return NewToolResultWithData(message, data), nil
}

func (tc *ToolsCatalog) searchVectors(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	if err := args.Require("collection_name", "vectors"); err != nil {
		return nil, err
	}

	dbName := args.GetString("database", "default")
	collectionName := args.GetString("collection_name", "")
	vectors := GetFloatSlice(args, "vectors")
	limit := args.GetInt("limit", DefaultSearchLimit)
	filter := args.GetString("filter", "")
	outputFields := GetStringSlice(args, "output_fields")

	if len(vectors) == 0 {
		return nil, fmt.Errorf(ErrVectorsRequired)
	}

	// Simplified search - assumes single vector search
	searchReq := &milvuspb.SearchRequest{
		DbName:         dbName,
		CollectionName: collectionName,
		Dsl:            filter,
		DslType:        commonpb.DslType_BoolExprV1,
		OutputFields:   outputFields,
		SearchParams: []*commonpb.KeyValuePair{
			{Key: "anns_field", Value: "vector"},
			{Key: "topk", Value: fmt.Sprintf("%d", limit)},
			{Key: "metric_type", Value: "L2"},
			{Key: "params", Value: `{"nprobe": 10}`},
		},
		Nq: 1,
	}

	resp, err := tc.proxy.Search(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	// Simplified result formatting
	data := map[string]interface{}{
		"results":         resp.Results,
		"collection_name": collectionName,
		"database":        dbName,
	}

	message := fmt.Sprintf(MsgSearchSuccess, len(resp.Results.Ids.GetIntId().GetData()))
	return NewToolResultWithData(message, data), nil
}

func (tc *ToolsCatalog) insertData(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Simplified implementation
	return NewToolResult(MsgInsertPlaceholder), nil
}

func (tc *ToolsCatalog) queryData(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Simplified implementation
	return NewToolResult(MsgQueryPlaceholder), nil
}

func (tc *ToolsCatalog) deleteData(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Simplified implementation
	return NewToolResult(MsgDeletePlaceholder), nil
}

func (tc *ToolsCatalog) createIndex(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Simplified implementation
	return NewToolResult(MsgCreateIndexPlaceholder), nil
}

func (tc *ToolsCatalog) describeIndex(ctx context.Context, args ToolArgs) (*ToolResult, error) {
	// Simplified implementation
	return NewToolResult(MsgDescribeIndexPlaceholder), nil
}

// Schema definitions

// Helper function to create simple collection name schema
func (tc *ToolsCatalog) simpleCollectionSchema() *ToolSchema {
	return &ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaParam{
			"database": {
				Type:        "string",
				Description: ParamDatabaseDescription,
				Default:     "default",
			},
			"collection_name": {
				Type:        "string",
				Description: ParamCollectionNameDescription,
			},
		},
		Required: []string{"collection_name"},
	}
}

func (tc *ToolsCatalog) schemaForListCollections() *ToolSchema {
	return &ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaParam{
			"database": {
				Type:        "string",
				Description: ParamDatabaseDefaultDesc,
				Default:     "default",
			},
		},
	}
}

func (tc *ToolsCatalog) schemaForCreateCollection() *ToolSchema {
	min1 := 1
	max255 := 255
	minDim := float64(1)
	maxDim := float64(32768)

	// 基于MCP 2025-06-18规范创建schema
	return NewToolSchema().
		AddParameter("database", &SchemaParam{
			Type:        "string",
			Description: ParamDatabaseDescription,
			Default:     "default",
		}).
		AddParameter("collection_name", &SchemaParam{
			Type:        "string",
			Description: ParamCollectionNameCreateDescription,
			MinLength:   &min1,
			MaxLength:   &max255,
		}).
		AddParameter("dimension", &SchemaParam{
			Type:        "integer",
			Description: ParamDimensionDescription,
			Minimum:     &minDim,
			Maximum:     &maxDim,
		}).
		AddParameter("metric_type", &SchemaParam{
			Type:        "string",
			Description: ParamMetricTypeDescription,
			Enum:        []interface{}{"L2", "IP", "COSINE"},
			Default:     "L2",
		}).
		AddRequired("collection_name", "dimension")
}

func (tc *ToolsCatalog) schemaForDescribeCollection() *ToolSchema {
	return &ToolSchema{
		Type: "object",
		Properties: map[string]*SchemaParam{
			"database": {
				Type:        "string",
				Description: "Database name",
				Default:     "default",
			},
			"collection_name": {
				Type:        "string",
				Description: "Name of the collection",
			},
		},
		Required: []string{"collection_name"},
	}
}

// Output Schema definitions

func (tc *ToolsCatalog) outputSchemaForListCollections() *ToolSchema {
	return NewToolSchema().
		AddParameter("collections", &SchemaParam{
			Type:        "array",
			Description: OutputCollectionsDescription,
			Items: &SchemaParam{
				Type: "string",
			},
		}).
		AddParameter("database", &SchemaParam{
			Type:        "string",
			Description: OutputDatabaseDescription,
		}).
		AddRequired("collections", "database")
}

func (tc *ToolsCatalog) outputSchemaForCreateCollection() *ToolSchema {
	return NewToolSchema().
		AddParameter("collection_name", &SchemaParam{
			Type:        "string",
			Description: OutputCollectionNameDescription,
		}).
		AddParameter("database", &SchemaParam{
			Type:        "string",
			Description: "Database name",
		}).
		AddParameter("dimension", &SchemaParam{
			Type:        "integer",
			Description: OutputDimensionDescription,
		}).
		AddParameter("metric_type", &SchemaParam{
			Type:        "string",
			Description: OutputMetricTypeDescription,
		}).
		AddParameter("status", &SchemaParam{
			Type:        "string",
			Description: OutputStatusDescription,
		}).
		AddRequired("collection_name", "database", "dimension", "metric_type", "status")
}

// 其他复杂schema方法已简化为直接使用simpleCollectionSchema()
