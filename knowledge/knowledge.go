package knowledge

import (
	"context"
	"errors"
)

// Entity represents an entity in the knowledge graph
type Entity struct {
	ID         string                 // unique identifier of the entity
	Type       string                 // entity type
	Name       string                 // entity name
	Properties map[string]interface{} // entity properties
}

// Relation represents a relationship between entities
type Relation struct {
	ID         string                 // unique identifier of the relation
	Type       string                 // relation type
	SourceID   string                 // source entity ID
	TargetID   string                 // target entity ID
	Properties map[string]interface{} // relation properties
}

// Query represents query conditions for the knowledge graph
type Query struct {
	EntityTypes   []string               // entity types to query
	RelationTypes []string               // relation types to query
	Properties    map[string]interface{} // property filter conditions
	Limit         int                    // result limit
}

// Graph is the basic operation interface for knowledge graph
type Graph interface {
	// AddEntity adds an entity to the knowledge graph
	AddEntity(ctx context.Context, entity Entity) (string, error)

	// AddRelation adds a relation to the knowledge graph
	AddRelation(ctx context.Context, relation Relation) (string, error)

	// GetEntity gets an entity by ID
	GetEntity(ctx context.Context, id string) (Entity, error)

	// GetRelation gets a relation by ID
	GetRelation(ctx context.Context, id string) (Relation, error)

	// QueryEntities queries entities that match the conditions
	QueryEntities(ctx context.Context, query Query) ([]Entity, error)

	// QueryRelations queries relations that match the conditions
	QueryRelations(ctx context.Context, query Query) ([]Relation, error)

	// GetRelatedEntities gets entities related to a specific entity
	GetRelatedEntities(ctx context.Context, entityID string, relationType string) ([]Entity, error)

	// DeleteEntity deletes an entity (and its related relations)
	DeleteEntity(ctx context.Context, id string) error

	// DeleteRelation deletes a relation
	DeleteRelation(ctx context.Context, id string) error

	// Clear clears the knowledge graph
	Clear(ctx context.Context) error
}

// ErrEntityNotFound entity not found error
var ErrEntityNotFound = errors.New("entity not found")

// ErrRelationNotFound relation not found error
var ErrRelationNotFound = errors.New("relation not found")

// ErrInvalidInput invalid input error
var ErrInvalidInput = errors.New("invalid input parameters")

// ErrDuplicateID duplicate ID error
var ErrDuplicateID = errors.New("ID already exists")
