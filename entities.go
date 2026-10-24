package twelvelabs

import (
	"context"
	"net/http"
	"net/url"
)

// Entity status values.
const (
	EntityStatusProcessing = "processing"
	EntityStatusReady      = "ready"
)

// EntitiesService manages entity collections and the people in them.
type EntitiesService struct {
	client *Client
}

// EntityCollection is a named group of entities.
type EntityCollection struct {
	ID          string `json:"_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// Entity is a person or object identified across one or more assets.
type Entity struct {
	ID                 string         `json:"_id"`
	EntityCollectionID string         `json:"entity_collection_id,omitempty"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	AssetIDs           []string       `json:"asset_ids,omitempty"`
	Status             string         `json:"status,omitempty"`
	CreatedAt          string         `json:"created_at,omitempty"`
	UpdatedAt          string         `json:"updated_at,omitempty"`
}

// EntityCollectionCreateRequest creates a collection.
type EntityCollectionCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// EntityCollectionUpdateRequest updates a collection. Empty fields are left
// unchanged.
type EntityCollectionUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// EntityCreateRequest creates an entity from one or more assets.
type EntityCreateRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	AssetIDs    []string       `json:"asset_ids"`
}

// EntityUpdateRequest updates an entity. Metadata, when set, replaces the
// existing metadata entirely.
type EntityUpdateRequest struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// EntityAssetsRequest adds or removes assets for an entity.
type EntityAssetsRequest struct {
	AssetIDs []string `json:"asset_ids"`
}

// EntityBulkCreateRequest creates up to many entities in one call.
type EntityBulkCreateRequest struct {
	Entities []EntityCreateRequest `json:"entities"`
}

// BulkEntityResult is one successfully created entity of a bulk request.
type BulkEntityResult struct {
	ID     string `json:"_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// BulkEntityError is one failed entity of a bulk request.
type BulkEntityError struct {
	EntityIndex int    `json:"entity_index"`
	EntityName  string `json:"entity_name"`
	ErrorReason string `json:"error_reason"`
}

// EntityBulkCreateResponse reports the outcome of a bulk create.
type EntityBulkCreateResponse struct {
	SuccessCount int                `json:"success_count"`
	FailedCount  int                `json:"failed_count"`
	Entities     []BulkEntityResult `json:"entities"`
	Errors       []BulkEntityError  `json:"errors"`
}

// EntityCollectionListOptions filters ListCollections.
type EntityCollectionListOptions struct {
	Pagination
	Name       string
	SortBy     string
	SortOption string
}

// EntityListOptions filters ListEntities.
type EntityListOptions struct {
	Pagination
	Name       string
	Status     string
	SortBy     string
	SortOption string
}

// ListCollections returns entity collections.
func (s *EntitiesService) ListCollections(ctx context.Context, options *EntityCollectionListOptions) (*Page[EntityCollection], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		setIfNotEmpty(query, "name", options.Name)
		setIfNotEmpty(query, "sort_by", options.SortBy)
		setIfNotEmpty(query, "sort_option", options.SortOption)
	}

	var page Page[EntityCollection]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/entity-collections", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateCollection creates an entity collection.
func (s *EntitiesService) CreateCollection(ctx context.Context, request *EntityCollectionCreateRequest) (*EntityCollection, error) {
	ctx = ensureContext(ctx)
	if request == nil || request.Name == "" {
		return nil, validationError("Name", "is required")
	}
	var collection EntityCollection
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/entity-collections", nil, request, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

// GetCollection retrieves an entity collection.
func (s *EntitiesService) GetCollection(ctx context.Context, collectionID string) (*EntityCollection, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	var collection EntityCollection
	path := "/entity-collections/" + escapePath(collectionID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

// UpdateCollection renames or re-describes an entity collection. Only the
// fields that are set are changed.
func (s *EntitiesService) UpdateCollection(ctx context.Context, collectionID string, request *EntityCollectionUpdateRequest) (*EntityCollection, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	if request == nil || (request.Name == "" && request.Description == "") {
		return nil, validationError("request", "set Name, Description, or both")
	}
	var collection EntityCollection
	path := "/entity-collections/" + escapePath(collectionID)
	if err := s.client.transport.doJSON(ctx, http.MethodPatch, path, nil, request, &collection); err != nil {
		return nil, err
	}
	return &collection, nil
}

// DeleteCollection removes an entity collection and its entities.
func (s *EntitiesService) DeleteCollection(ctx context.Context, collectionID string) error {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return validationError("collectionID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/entity-collections/"+escapePath(collectionID), nil, nil, nil)
}

// ListEntities returns the entities of a collection.
func (s *EntitiesService) ListEntities(ctx context.Context, collectionID string, options *EntityListOptions) (*Page[Entity], error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		setIfNotEmpty(query, "name", options.Name)
		setIfNotEmpty(query, "status", options.Status)
		setIfNotEmpty(query, "sort_by", options.SortBy)
		setIfNotEmpty(query, "sort_option", options.SortOption)
	}

	var page Page[Entity]
	path := "/entity-collections/" + escapePath(collectionID) + "/entities"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateEntity adds an entity to a collection. The entity starts as processing
// while the platform matches the person in the supplied assets.
func (s *EntitiesService) CreateEntity(ctx context.Context, collectionID string, request *EntityCreateRequest) (*Entity, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	if request == nil || request.Name == "" {
		return nil, validationError("Name", "is required")
	}
	if len(request.AssetIDs) == 0 {
		return nil, validationError("AssetIDs", "at least one asset is required")
	}

	var entity Entity
	path := "/entity-collections/" + escapePath(collectionID) + "/entities"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, request, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// BulkCreateEntities adds many entities at once. The response reports partial
// success, so callers should check both FailedCount and Errors.
func (s *EntitiesService) BulkCreateEntities(ctx context.Context, collectionID string, request *EntityBulkCreateRequest) (*EntityBulkCreateResponse, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	if request == nil || len(request.Entities) == 0 {
		return nil, validationError("Entities", "at least one entity is required")
	}
	for i, entity := range request.Entities {
		if entity.Name == "" {
			return nil, validationError("Entities", "entity %d has no Name", i)
		}
		if len(entity.AssetIDs) == 0 {
			return nil, validationError("Entities", "entity %d has no AssetIDs", i)
		}
	}

	var response EntityBulkCreateResponse
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/bulk"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetEntity retrieves an entity.
func (s *EntitiesService) GetEntity(ctx context.Context, collectionID, entityID string) (*Entity, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	if entityID == "" {
		return nil, validationError("entityID", "is required")
	}
	var entity Entity
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/" + escapePath(entityID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// UpdateEntity updates an entity. Metadata, when set, replaces the existing
// metadata.
func (s *EntitiesService) UpdateEntity(ctx context.Context, collectionID, entityID string, request *EntityUpdateRequest) (*Entity, error) {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return nil, validationError("collectionID", "is required")
	}
	if entityID == "" {
		return nil, validationError("entityID", "is required")
	}
	if request == nil {
		return nil, validationError("request", "is required")
	}
	var entity Entity
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/" + escapePath(entityID)
	if err := s.client.transport.doJSON(ctx, http.MethodPatch, path, nil, request, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// DeleteEntity removes an entity from a collection.
func (s *EntitiesService) DeleteEntity(ctx context.Context, collectionID, entityID string) error {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return validationError("collectionID", "is required")
	}
	if entityID == "" {
		return validationError("entityID", "is required")
	}
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/" + escapePath(entityID)
	return s.client.transport.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// AddEntityAssets associates more assets with an entity.
func (s *EntitiesService) AddEntityAssets(ctx context.Context, collectionID, entityID string, assetIDs ...string) error {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return validationError("collectionID", "is required")
	}
	if entityID == "" {
		return validationError("entityID", "is required")
	}
	if len(assetIDs) == 0 {
		return validationError("AssetIDs", "at least one asset is required")
	}
	body := EntityAssetsRequest{AssetIDs: assetIDs}
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/" + escapePath(entityID) + "/assets"
	return s.client.transport.doJSON(ctx, http.MethodPost, path, nil, &body, nil)
}

// RemoveEntityAssets dissociates assets from an entity.
func (s *EntitiesService) RemoveEntityAssets(ctx context.Context, collectionID, entityID string, assetIDs ...string) error {
	ctx = ensureContext(ctx)
	if collectionID == "" {
		return validationError("collectionID", "is required")
	}
	if entityID == "" {
		return validationError("entityID", "is required")
	}
	if len(assetIDs) == 0 {
		return validationError("AssetIDs", "at least one asset is required")
	}
	body := EntityAssetsRequest{AssetIDs: assetIDs}
	path := "/entity-collections/" + escapePath(collectionID) + "/entities/" + escapePath(entityID) + "/assets"
	return s.client.transport.doJSON(ctx, http.MethodDelete, path, nil, &body, nil)
}
