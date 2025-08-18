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

// Tool descriptions and titles
const (
	// Collection management tools
	ToolCollectionListName        = "collection.list"
	ToolCollectionListTitle       = "List Collections"
	ToolCollectionListDescription = "List all collections in a database"

	ToolCollectionCreateName        = "collection.create"
	ToolCollectionCreateTitle       = "Create Collection"
	ToolCollectionCreateDescription = "Create a new vector collection with simplified parameters"

	ToolCollectionDescribeName        = "collection.describe"
	ToolCollectionDescribeTitle       = "Describe Collection"
	ToolCollectionDescribeDescription = "Get detailed information about a collection"

	ToolCollectionDropName        = "collection.drop"
	ToolCollectionDropTitle       = "Drop Collection"
	ToolCollectionDropDescription = "Delete a collection and all its data"

	// Data operation tools
	ToolDataInsertName        = "data.insert"
	ToolDataInsertTitle       = "Insert Data"
	ToolDataInsertDescription = "Insert vectors and associated data into a collection"

	ToolDataSearchName        = "data.search"
	ToolDataSearchTitle       = "Vector Search"
	ToolDataSearchDescription = "Search for similar vectors in a collection"

	ToolDataQueryName        = "data.query"
	ToolDataQueryTitle       = "Query Data"
	ToolDataQueryDescription = "Query data using scalar filters"

	ToolDataDeleteName        = "data.delete"
	ToolDataDeleteTitle       = "Delete Data"
	ToolDataDeleteDescription = "Delete entities from a collection by primary key"

	// Index management tools
	ToolIndexCreateName        = "index.create"
	ToolIndexCreateTitle       = "Create Index"
	ToolIndexCreateDescription = "Create an index on a vector field"

	ToolIndexDescribeName        = "index.describe"
	ToolIndexDescribeTitle       = "Describe Index"
	ToolIndexDescribeDescription = "Get information about indexes on a collection"
)

// Parameter descriptions - Input parameters
const (
	// Common parameters
	ParamDatabaseDescription       = "Database name"
	ParamDatabaseDefaultDesc       = "Database name (default: 'default')"
	ParamCollectionNameDescription = "Name of the collection"

	// Collection creation parameters
	ParamCollectionNameCreateDescription = "Name of the collection to create"
	ParamDimensionDescription            = "Dimension of vectors"
	ParamMetricTypeDescription           = "Distance metric type"
)

// Parameter descriptions - Output parameters
const (
	// Collection list output
	OutputCollectionsDescription = "List of collection names"
	OutputDatabaseDescription    = "Database name"

	// Collection creation output
	OutputCollectionNameDescription = "Name of the created collection"
	OutputDimensionDescription      = "Vector dimension"
	OutputMetricTypeDescription     = "Distance metric type"
	OutputStatusDescription         = "Operation status"
)

// Success messages
const (
	MsgCollectionListSuccess     = "Found %d collections in database '%s'"
	MsgCollectionCreateSuccess   = "Collection '%s' created successfully with dimension %d"
	MsgCollectionDescribeSuccess = "Collection '%s' has %d fields"
	MsgCollectionDropSuccess     = "Collection '%s' dropped successfully"
	MsgSearchSuccess             = "Search completed with %d results"
)

// Placeholder messages for unimplemented features
const (
	MsgInsertPlaceholder        = "Insert operation would be implemented here"
	MsgQueryPlaceholder         = "Query operation would be implemented here"
	MsgDeletePlaceholder        = "Delete operation would be implemented here"
	MsgCreateIndexPlaceholder   = "Create index operation would be implemented here"
	MsgDescribeIndexPlaceholder = "Describe index operation would be implemented here"
)

// Error messages
const (
	ErrDimensionPositive = "dimension must be positive, got %d"
	ErrVectorsRequired   = "vectors cannot be empty"
)
