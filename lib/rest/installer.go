package rest

import (
	"net/http"
	"strings"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/runtime"
)

// APIInstaller registers routes for an APIGroupInfo on a WebService.
type APIInstaller struct {
	group      *APIGroupInfo
	ws         *WebService
	serializer runtime.NegotiatedSerializer
}

// Install registers all resource and sub-resource routes.
func (i *APIInstaller) Install() {
	for _, res := range i.group.Resources {
		i.installResource(res)
	}
}

func (i *APIInstaller) installResource(res ResourceInfo) {
	idParam := defaultIDParam(res.Name)

	basePath := "/" + res.Name
	itemPath := basePath + "/{" + idParam + "}"

	storage := res.Storage

	// POST /{resources}
	if s, ok := storage.(Creator); ok {
		handler := i.createHandler(s)
		i.ws.Route(i.ws.POST(basePath).To(handler))
		logger.Infof("  POST   %s%s", i.ws.RootPath(), basePath)
	}

	// GET /{resources}
	if s, ok := storage.(Lister); ok {
		handler := i.listHandler(s)
		i.ws.Route(i.ws.GET(basePath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), basePath)
	}

	// GET /{resources}/{id}
	if s, ok := storage.(Getter); ok {
		handler := i.getHandler(s)
		i.ws.Route(i.ws.GET(itemPath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), itemPath)
	}

	// PUT /{resources}/{id}
	if s, ok := storage.(Updater); ok {
		handler := i.updateHandler(s)
		i.ws.Route(i.ws.PUT(itemPath).To(handler))
		logger.Infof("  PUT    %s%s", i.ws.RootPath(), itemPath)
	}

	// PATCH /{resources}/{id}
	if s, ok := storage.(Patcher); ok {
		handler := i.patchHandler(s)
		i.ws.Route(i.ws.PATCH(itemPath).To(handler))
		logger.Infof("  PATCH  %s%s", i.ws.RootPath(), itemPath)
	}

	// DELETE /{resources}/{id}
	if s, ok := storage.(Deleter); ok {
		handler := i.deleteHandler(s)
		i.ws.Route(i.ws.DELETE(itemPath).To(handler))
		logger.Infof("  DELETE %s%s", i.ws.RootPath(), itemPath)
	}

	// DELETE /{resources} (collection)
	if s, ok := storage.(CollectionDeleter); ok {
		handler := i.deleteCollectionHandler(s)
		i.ws.Route(i.ws.DELETE(basePath).To(handler))
		logger.Infof("  DELETE %s%s (collection)", i.ws.RootPath(), basePath)
	}

	// Actions on item
	for _, action := range res.Actions {
		i.installAction(itemPath, action)
	}

	// Sub-resources
	for _, sub := range res.SubResources {
		i.installSubResource(itemPath, sub)
	}
}

func (i *APIInstaller) installSubResource(parentItemPath string, sub ResourceInfo) {
	subIDParam := defaultIDParam(sub.Name)

	basePath := parentItemPath + "/" + sub.Name
	itemPath := basePath + "/{" + subIDParam + "}"

	storage := sub.Storage

	if s, ok := storage.(Creator); ok {
		handler := i.createHandler(s)
		i.ws.Route(i.ws.POST(basePath).To(handler))
		logger.Infof("  POST   %s%s", i.ws.RootPath(), basePath)
	}

	if s, ok := storage.(Lister); ok {
		handler := i.listHandler(s)
		i.ws.Route(i.ws.GET(basePath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), basePath)
	}

	if s, ok := storage.(Getter); ok {
		handler := i.getHandler(s)
		i.ws.Route(i.ws.GET(itemPath).To(handler))
		logger.Infof("  GET    %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Updater); ok {
		handler := i.updateHandler(s)
		i.ws.Route(i.ws.PUT(itemPath).To(handler))
		logger.Infof("  PUT    %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Patcher); ok {
		handler := i.patchHandler(s)
		i.ws.Route(i.ws.PATCH(itemPath).To(handler))
		logger.Infof("  PATCH  %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(Deleter); ok {
		handler := i.deleteHandler(s)
		i.ws.Route(i.ws.DELETE(itemPath).To(handler))
		logger.Infof("  DELETE %s%s", i.ws.RootPath(), itemPath)
	}

	if s, ok := storage.(CollectionDeleter); ok {
		handler := i.deleteCollectionHandler(s)
		i.ws.Route(i.ws.DELETE(basePath).To(handler))
		logger.Infof("  DELETE %s%s (collection)", i.ws.RootPath(), basePath)
	}

	// Actions on sub-resource item
	for _, action := range sub.Actions {
		i.installAction(itemPath, action)
	}

	// Recursive sub-resources
	for _, nested := range sub.SubResources {
		i.installSubResource(itemPath, nested)
	}
}

// installAction registers a custom action route on a resource item.
func (i *APIInstaller) installAction(parentItemPath string, action ActionInfo) {
	actionPath := parentItemPath + "/" + action.Name
	statusCode := action.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	handler := Handle(i.serializer, statusCode, action.Handler)
	i.ws.Route(i.ws.METHOD(action.Method, actionPath).To(handler))
	logger.Infof("  %s %s%s (action)", action.Method, i.ws.RootPath(), actionPath)
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
func (i *APIInstaller) createHandler(storage Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		body, err := readBody(req)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		var into runtime.Object
		if oc, ok := storage.(ObjectCreator); ok {
			into = oc.NewObject()
		}

		obj, err := DecodeBody(i.serializer, req, body, into)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		result, err := storage.Create(ctx, obj, &CreateOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteObjectNegotiated(i.serializer, w, req, http.StatusCreated, result)
	}
}

// listHandler returns an http.HandlerFunc for GET (list).
func (i *APIInstaller) listHandler(storage Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)
		options := ParseListOptions(req.URL.Query())
		options.PathParams = params

		result, err := storage.List(ctx, options)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteObjectNegotiated(i.serializer, w, req, http.StatusOK, result)
	}
}

// getHandler returns an http.HandlerFunc for GET (single resource).
func (i *APIInstaller) getHandler(storage Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		result, err := storage.Get(ctx, &GetOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteObjectNegotiated(i.serializer, w, req, http.StatusOK, result)
	}
}

// updateHandler returns an http.HandlerFunc for PUT (full update).
func (i *APIInstaller) updateHandler(storage Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		body, err := readBody(req)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		var into runtime.Object
		if oc, ok := storage.(ObjectCreator); ok {
			into = oc.NewObject()
		}

		obj, err := DecodeBody(i.serializer, req, body, into)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		result, err := storage.Update(ctx, obj, &UpdateOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteObjectNegotiated(i.serializer, w, req, http.StatusOK, result)
	}
}

// patchHandler returns an http.HandlerFunc for PATCH (partial update).
func (i *APIInstaller) patchHandler(storage Patcher) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		body, err := readBody(req)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		var into runtime.Object
		if oc, ok := storage.(ObjectCreator); ok {
			into = oc.NewObject()
		}

		obj, err := DecodeBody(i.serializer, req, body, into)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		result, err := storage.Patch(ctx, obj, &PatchOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteObjectNegotiated(i.serializer, w, req, http.StatusOK, result)
	}
}

// deleteHandler returns an http.HandlerFunc for DELETE (single resource).
func (i *APIInstaller) deleteHandler(storage Deleter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		err := storage.Delete(ctx, &DeleteOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// deleteCollectionHandler returns an http.HandlerFunc for DELETE (collection).
func (i *APIInstaller) deleteCollectionHandler(storage CollectionDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		body, err := readBody(req)
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		var deleteReq DeleteCollectionRequest
		if err := jsonUnmarshal(body, &deleteReq); err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		if len(deleteReq.IDs) == 0 {
			handleError(i.serializer, errNoIDs(), w, req)
			return
		}

		result, err := storage.DeleteCollection(ctx, deleteReq.IDs, &DeleteOptions{})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		WriteRawJSON(w, http.StatusOK, result)
	}
}
