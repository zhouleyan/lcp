package rest

import (
	"net/http"
	"strings"

	"lcp.io/lcp/lib/logger"
)

// APIInstaller registers routes for an APIGroupInfo on a WebService.
type APIInstaller struct {
	group *APIGroupInfo
	ws    *WebService
	scope *RequestScope
}

// NewAPIInstaller creates a new installer for the given group, web service and scope.
func NewAPIInstaller(group *APIGroupInfo, ws *WebService, scope *RequestScope) *APIInstaller {
	return &APIInstaller{group: group, ws: ws, scope: scope}
}

// Install registers all resource and sub-resource routes.
func (i *APIInstaller) Install() {
	for _, res := range i.group.Resources {
		i.installResource(res)
	}
}

func (i *APIInstaller) installResource(res ResourceInfo) {
	idParam := res.IDParam
	if idParam == "" {
		idParam = defaultIDParam(res.Name)
	}

	basePath := "/" + res.Name
	itemPath := basePath + "/{" + idParam + "}"

	storage := res.Storage

	// POST /{resources}
	if s, ok := storage.(Creator); ok {
		handler := createHandler(i.scope, s)
		i.ws.Route(i.ws.POST(basePath).To(handler))
		logger.Infof("  POST   %s%s", i.ws.RootPath(), basePath)
	}

	// GET /{resources}
	if s, ok := storage.(Lister); ok {
		handler := listHandler(i.scope, s)
		i.ws.Route(i.ws.GET(basePath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), basePath)
	}

	// GET /{resources}/{id}
	if s, ok := storage.(Getter); ok {
		handler := getHandler(i.scope, s, idParam)
		i.ws.Route(i.ws.GET(itemPath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), itemPath)
	}

	// PUT /{resources}/{id}
	if s, ok := storage.(Updater); ok {
		handler := updateHandler(i.scope, s, idParam)
		i.ws.Route(i.ws.PUT(itemPath).To(handler))
		logger.Infof("  PUT    %s%s", i.ws.RootPath(), itemPath)
	}

	// PATCH /{resources}/{id}
	if s, ok := storage.(Patcher); ok {
		handler := patchHandler(i.scope, s, idParam)
		i.ws.Route(i.ws.PATCH(itemPath).To(handler))
		logger.Infof("  PATCH  %s%s", i.ws.RootPath(), itemPath)
	}

	// DELETE /{resources}/{id}
	if s, ok := storage.(Deleter); ok {
		handler := deleteHandler(i.scope, s, idParam)
		i.ws.Route(i.ws.DELETE(itemPath).To(handler))
		logger.Infof("  DELETE %s%s", i.ws.RootPath(), itemPath)
	}

	// DELETE /{resources} (collection)
	if s, ok := storage.(CollectionDeleter); ok {
		handler := deleteCollectionHandler(i.scope, s)
		i.ws.Route(i.ws.DELETE(basePath).To(handler))
		logger.Infof("  DELETE %s%s (collection)", i.ws.RootPath(), basePath)
	}

	// Sub-resources
	for _, sub := range res.SubResources {
		i.installSubResource(itemPath, sub)
	}
}

func (i *APIInstaller) installSubResource(parentItemPath string, sub SubResourceInfo) {
	subIDParam := sub.IDParam
	if subIDParam == "" {
		subIDParam = defaultIDParam(sub.Name)
	}

	basePath := parentItemPath + "/" + sub.Name
	itemPath := basePath + "/{" + subIDParam + "}"

	storage := sub.Storage

	if s, ok := storage.(Creator); ok {
		handler := createHandler(i.scope, s)
		i.ws.Route(i.ws.POST(basePath).To(handler))
		logger.Infof("  POST   %s%s", i.ws.RootPath(), basePath)
	}

	if s, ok := storage.(Lister); ok {
		handler := listHandler(i.scope, s)
		i.ws.Route(i.ws.GET(basePath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), basePath)
	}

	if s, ok := storage.(Getter); ok {
		handler := getHandler(i.scope, s, subIDParam)
		i.ws.Route(i.ws.GET(itemPath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Updater); ok {
		handler := updateHandler(i.scope, s, subIDParam)
		i.ws.Route(i.ws.PUT(itemPath).To(handler))
		logger.Infof("  PUT    %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Patcher); ok {
		handler := patchHandler(i.scope, s, subIDParam)
		i.ws.Route(i.ws.PATCH(itemPath).To(handler))
		logger.Infof("  PATCH  %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Deleter); ok {
		handler := deleteHandler(i.scope, s, subIDParam)
		i.ws.Route(i.ws.DELETE(itemPath).To(handler))
		logger.Infof("  DELETE %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(CollectionDeleter); ok {
		handler := deleteCollectionHandler(i.scope, s)
		i.ws.Route(i.ws.DELETE(basePath).To(handler))
		logger.Infof("  DELETE %s%s (collection)", i.ws.RootPath(), basePath)
	}
}

// defaultIDParam derives an ID parameter name from a plural resource name.
// "users" -> "userId", "namespaces" -> "namespaceId", "members" -> "memberId"
func defaultIDParam(plural string) string {
	singular := strings.TrimSuffix(plural, "s")
	if strings.HasSuffix(singular, "se") {
		// e.g. "namespaces" -> "namespace" (not "namespacese" -> trim "s")
		singular = strings.TrimSuffix(plural, "es")
		if singular == "" {
			singular = plural
		}
	}
	// Simple heuristic
	singular = strings.TrimSuffix(plural, "s")
	return singular + "Id"
}

// createHandler returns an http.HandlerFunc for POST (create).
func createHandler(scope *RequestScope, storage Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		body, err := readBody(req)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		obj, err := DecodeBody(scope.Serializer, req, body, nil)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		result, err := storage.Create(ctx, obj, &CreateOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusCreated, result)
	}
}

// listHandler returns an http.HandlerFunc for GET (list).
func listHandler(scope *RequestScope, storage Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		options := ParseListOptions(req.URL.Query())

		result, err := storage.List(ctx, options)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}

// getHandler returns an http.HandlerFunc for GET (single resource).
func getHandler(scope *RequestScope, storage Getter, idKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params[idKey]
		if id == "" {
			scope.err(errMissingID(idKey), w, req)
			return
		}

		result, err := storage.Get(ctx, id)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}

// updateHandler returns an http.HandlerFunc for PUT (full update).
func updateHandler(scope *RequestScope, storage Updater, idKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params[idKey]
		if id == "" {
			scope.err(errMissingID(idKey), w, req)
			return
		}

		body, err := readBody(req)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		obj, err := DecodeBody(scope.Serializer, req, body, nil)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		result, err := storage.Update(ctx, id, obj, &UpdateOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}

// patchHandler returns an http.HandlerFunc for PATCH (partial update).
func patchHandler(scope *RequestScope, storage Patcher, idKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params[idKey]
		if id == "" {
			scope.err(errMissingID(idKey), w, req)
			return
		}

		body, err := readBody(req)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		obj, err := DecodeBody(scope.Serializer, req, body, nil)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		result, err := storage.Patch(ctx, id, obj, &PatchOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteObjectNegotiated(scope.Serializer, w, req, http.StatusOK, result)
	}
}

// deleteHandler returns an http.HandlerFunc for DELETE (single resource).
func deleteHandler(scope *RequestScope, storage Deleter, idKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := pathParamsFromRequest(req)

		id := params[idKey]
		if id == "" {
			scope.err(errMissingID(idKey), w, req)
			return
		}

		err := storage.Delete(ctx, id, &DeleteOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// deleteCollectionHandler returns an http.HandlerFunc for DELETE (collection).
func deleteCollectionHandler(scope *RequestScope, storage CollectionDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		body, err := readBody(req)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		var deleteReq DeleteCollectionRequest
		if err := jsonUnmarshal(body, &deleteReq); err != nil {
			scope.err(err, w, req)
			return
		}

		if len(deleteReq.IDs) == 0 {
			scope.err(errNoIDs(), w, req)
			return
		}

		result, err := storage.DeleteCollection(ctx, deleteReq.IDs, &DeleteOptions{})
		if err != nil {
			scope.err(err, w, req)
			return
		}

		WriteRawJSON(w, http.StatusOK, result)
	}
}
