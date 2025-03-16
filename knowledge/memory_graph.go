package knowledge

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// MemoryGraph is an in-memory implementation of the Graph interface
type MemoryGraph struct {
	entities  map[string]Entity
	relations map[string]Relation
	mutex     sync.RWMutex
}

func NewMemoryGraph() *MemoryGraph {
	return &MemoryGraph{
		entities:  make(map[string]Entity),
		relations: make(map[string]Relation),
	}
}

// AddEntity adds an entity to the knowledge graph
func (g *MemoryGraph) AddEntity(ctx context.Context, entity Entity) (string, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if entity.ID == "" {
		entity.ID = uuid.New().String()
	} else if _, exists := g.entities[entity.ID]; exists {
		return "", ErrDuplicateID
	}

	if entity.Properties == nil {
		entity.Properties = make(map[string]interface{})
	}

	g.entities[entity.ID] = entity
	return entity.ID, nil
}

// AddRelation adds a relation to the knowledge graph
func (g *MemoryGraph) AddRelation(ctx context.Context, relation Relation) (string, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// verify if source and target entities exist
	if _, exists := g.entities[relation.SourceID]; !exists {
		return "", ErrEntityNotFound
	}
	if _, exists := g.entities[relation.TargetID]; !exists {
		return "", ErrEntityNotFound
	}

	if relation.ID == "" {
		relation.ID = uuid.New().String()
	} else if _, exists := g.relations[relation.ID]; exists {
		return "", ErrDuplicateID
	}

	if relation.Properties == nil {
		relation.Properties = make(map[string]interface{})
	}

	g.relations[relation.ID] = relation
	return relation.ID, nil
}

// GetEntity gets an entity by ID
func (g *MemoryGraph) GetEntity(ctx context.Context, id string) (Entity, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	entity, exists := g.entities[id]
	if !exists {
		return Entity{}, ErrEntityNotFound
	}

	return entity, nil
}

// GetRelation gets a relation by ID
func (g *MemoryGraph) GetRelation(ctx context.Context, id string) (Relation, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	relation, exists := g.relations[id]
	if !exists {
		return Relation{}, ErrRelationNotFound
	}

	return relation, nil
}

// QueryEntities queries entities that match the conditions
func (g *MemoryGraph) QueryEntities(ctx context.Context, query Query) ([]Entity, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var results []Entity

	for _, entity := range g.entities {
		if matchesEntityQuery(entity, query) {
			results = append(results, entity)

			// stop if limit is set and reached
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	return results, nil
}

// QueryRelations queries relations that match the conditions
func (g *MemoryGraph) QueryRelations(ctx context.Context, query Query) ([]Relation, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var results []Relation

	for _, relation := range g.relations {
		if matchesRelationQuery(relation, query) {
			results = append(results, relation)

			// stop if limit is set and reached
			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	return results, nil
}

// GetRelatedEntities gets entities related to a specific entity
func (g *MemoryGraph) GetRelatedEntities(ctx context.Context, entityID string, relationType string) ([]Entity, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// verify if entity exists
	if _, exists := g.entities[entityID]; !exists {
		return nil, ErrEntityNotFound
	}

	var results []Entity
	var relatedIDs = make(map[string]bool)

	// find relations where this entity is the source
	for _, relation := range g.relations {
		if relation.SourceID == entityID && (relationType == "" || relation.Type == relationType) {
			relatedIDs[relation.TargetID] = true
		}
	}

	// find relations where this entity is the target
	for _, relation := range g.relations {
		if relation.TargetID == entityID && (relationType == "" || relation.Type == relationType) {
			relatedIDs[relation.SourceID] = true
		}
	}

	// collect related entities
	for id := range relatedIDs {
		if entity, exists := g.entities[id]; exists {
			results = append(results, entity)
		}
	}

	return results, nil
}

// DeleteEntity deletes an entity and all its related relations
func (g *MemoryGraph) DeleteEntity(ctx context.Context, id string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, exists := g.entities[id]; !exists {
		return ErrEntityNotFound
	}

	for relationID, relation := range g.relations {
		if relation.SourceID == id || relation.TargetID == id {
			delete(g.relations, relationID)
		}
	}

	// delete entity
	delete(g.entities, id)

	return nil
}

// DeleteRelation deletes a relation
func (g *MemoryGraph) DeleteRelation(ctx context.Context, id string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, exists := g.relations[id]; !exists {
		return ErrRelationNotFound
	}

	// delete relation
	delete(g.relations, id)

	return nil
}

// Clear clears the knowledge graph
func (g *MemoryGraph) Clear(ctx context.Context) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.entities = make(map[string]Entity)
	g.relations = make(map[string]Relation)

	return nil
}

// matchesEntityQuery checks if an entity matches the query conditions
func matchesEntityQuery(entity Entity, query Query) bool {
	if len(query.EntityTypes) > 0 {
		typeMatched := false
		for _, t := range query.EntityTypes {
			if entity.Type == t {
				typeMatched = true
				break
			}
		}
		if !typeMatched {
			return false
		}
	}

	for key, value := range query.Properties {
		if entityValue, exists := entity.Properties[key]; !exists || !matchesProperty(entityValue, value) {
			return false
		}
	}

	return true
}

// matchesRelationQuery checks if a relation matches the query conditions
func matchesRelationQuery(relation Relation, query Query) bool {
	if len(query.RelationTypes) > 0 {
		typeMatched := false
		for _, t := range query.RelationTypes {
			if relation.Type == t {
				typeMatched = true
				break
			}
		}
		if !typeMatched {
			return false
		}
	}

	for key, value := range query.Properties {
		if relationValue, exists := relation.Properties[key]; !exists || !matchesProperty(relationValue, value) {
			return false
		}
	}

	return true
}

// matchesProperty checks if property values match, supports basic string prefix matching
func matchesProperty(entityValue, queryValue interface{}) bool {
	// if types are the same, compare directly
	if entityValue == queryValue {
		return true
	}

	entityStr, entityOk := entityValue.(string)
	queryStr, queryOk := queryValue.(string)

	if entityOk && queryOk && strings.HasPrefix(entityStr, queryStr) {
		return true
	}

	return false
}
