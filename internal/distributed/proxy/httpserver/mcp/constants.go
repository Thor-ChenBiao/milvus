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

// Protocol headers and parameter keys
const (
	MCPHeaderProtocolVersion = "MCP-Protocol-Version"
	ParamProtocolVersion     = "protocolVersion"
)

// Common tool argument keys
const (
	ParamDatabaseKey       = "database"
	ParamCollectionNameKey = "collection_name"
	ParamDimensionKey      = "dimension"
	ParamMetricTypeKey     = "metric_type"
)

// Collection schema defaults
const (
	FieldPrimaryIDName           = "id"
	FieldVectorName              = "vector"
	TypeParamDimKey              = "dim"
	DefaultCollectionDescription = "Created by MCP"
)

// Index and search parameter keys/defaults
const (
	DefaultIndexName        = "vector_index"
	DefaultIndexType        = "IVF_FLAT"
	DefaultMetricType       = "L2"
	IndexParamIndexTypeKey  = "index_type"
	IndexParamMetricTypeKey = "metric_type"
	IndexParamParamsKey     = "params"
)

// RBAC object types and privileges
const (
	ObjectTypeDatabase     = "Database"
	ObjectTypeCollection   = "Collection"
	PrivShowCollections    = "ShowCollections"
	PrivCreateCollection   = "CreateCollection"
	PrivDescribeCollection = "DescribeCollection"
	PrivDropCollection     = "DropCollection"
	PrivInsert             = "Insert"
	PrivSearch             = "Search"
	PrivQuery              = "Query"
	PrivDelete             = "Delete"
	PrivCreateIndex        = "CreateIndex"
	PrivDescribeIndex      = "DescribeIndex"
)

// Log event names (concise and searchable)
const (
	LogEvtInitStart       = "mcp.initialize.start"
	LogEvtInitDone        = "mcp.initialize.done"
	LogEvtToolsListStart  = "mcp.tools.list.start"
	LogEvtToolsListDone   = "mcp.tools.list.done"
	LogEvtToolsCallStart  = "mcp.tools.call.start"
	LogEvtToolsCallUser   = "mcp.tools.call.user"
	LogEvtToolsCallDenied = "mcp.tools.call.denied"
	LogEvtToolsCallFailed = "mcp.tools.call.failed"
	LogEvtToolsCallDone   = "mcp.tools.call.done"
)
