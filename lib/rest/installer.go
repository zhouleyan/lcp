package rest

import (
	"fmt"
	"net/http"
	"strings"

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
	idParam := resolveIDParam(res)
	basePath := "/" + res.Name
	itemPath := basePath + "/{" + idParam + "}"
	i.registerRoutes(basePath, itemPath, res)
}

func (i *APIInstaller) installSubResource(parentItemPath string, sub ResourceInfo) {
	idParam := resolveIDParam(sub)
	basePath := parentItemPath + "/" + sub.Name
	itemPath := basePath + "/{" + idParam + "}"
	i.registerRoutes(basePath, itemPath, sub)
}

func resolveIDParam(res ResourceInfo) string {
	if res.IDParam != "" {
		return res.IDParam
	}
	return defaultIDParam(res.Name)
}

func (i *APIInstaller) registerRoutes(basePath, itemPath string, res ResourceInfo) {
	storage := res.Storage

	if s, ok := storage.(Creator); ok {
		i.ws.Route(i.ws.POST(basePath).To(i.createHandler(s)))
	}

	if s, ok := storage.(Lister); ok {
		i.ws.Route(i.ws.GET(basePath).To(i.listHandler(s)))
	}

	if s, ok := storage.(Getter); ok {
		i.ws.Route(i.ws.GET(itemPath).To(i.getHandler(s)))
	}

	if s, ok := storage.(Updater); ok {
		i.ws.Route(i.ws.PUT(itemPath).To(i.updateHandler(s)))
	}

	if s, ok := storage.(Patcher); ok {
		i.ws.Route(i.ws.PATCH(itemPath).To(i.patchHandler(s)))
	}

	if s, ok := storage.(Deleter); ok {
		i.ws.Route(i.ws.DELETE(itemPath).To(i.deleteHandler(s)))
	}

	if s, ok := storage.(CollectionDeleter); ok {
		i.ws.Route(i.ws.DELETE(basePath).To(i.deleteCollectionHandler(s)))
	}

	for _, action := range res.Actions {
		i.installAction(itemPath, action)
	}

	for _, verb := range res.CustomVerbs {
		i.installCustomVerb(itemPath, verb)
	}

	for _, sub := range res.SubResources {
		i.installSubResource(itemPath, sub)
	}
}

// installAction registers a custom action route on a resource item.
// Action handlers receive both path params and query params (via HandleAction).
func (i *APIInstaller) installAction(parentItemPath string, action ActionInfo) {
	if action.Handler != nil && action.WebSocketHandler != nil {
		panic(fmt.Sprintf("action %q: Handler and WebSocketHandler are mutually exclusive", action.Name))
	}
	actionPath := parentItemPath + "/" + action.Name
	if action.WebSocketHandler != nil {
		handler := HandleWebSocket(action.WebSocketHandler)
		method := action.Method
		if method == "" {
			method = "GET"
		}
		i.ws.Route(i.ws.METHOD(method, actionPath).To(handler))
		return
	}
	statusCode := action.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	handler := HandleAction(i.serializer, statusCode, action.Handler, i.group.APIVersion())
	i.ws.Route(i.ws.METHOD(action.Method, actionPath).To(handler))
}

// installCustomVerb registers a custom verb route on a resource item.
// Custom verbs use the format {itemPath}:{verbName}, e.g. /users/{userId}:workspaces
func (i *APIInstaller) installCustomVerb(parentItemPath string, verb CustomVerbInfo) {
	parts := strings.Split(strings.TrimPrefix(parentItemPath, "/"), "/")
	parts[len(parts)-1] = parts[len(parts)-1] + ":" + verb.Name
	verbPath := "/" + strings.Join(parts, "/")
	i.ws.Route(i.ws.GET(verbPath).To(i.listHandler(verb.Storage)))
}

// setAPIVersion sets the APIVersion field on a runtime.Object if it has a TypeMeta.
func setAPIVersion(obj runtime.Object, apiVersion string) {
	if obj == nil {
		return
	}
	if tm := obj.GetTypeMeta(); tm != nil {
		tm.APIVersion = apiVersion
	}
}

// defaultIDParam derives an ID parameter name from a plural resource name.
// "users" -> "userId", "namespaces" -> "namespaceId", "members" -> "memberId"
func defaultIDParam(plural string) string {
	var singular string
	if strings.HasSuffix(plural, "ses") || strings.HasSuffix(plural, "xes") || strings.HasSuffix(plural, "zes") {
		singular = strings.TrimSuffix(plural, "es")
	} else {
		singular = strings.TrimSuffix(plural, "s")
	}
	if singular == "" {
		singular = plural
	}
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

		setAPIVersion(result, i.group.APIVersion())
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

		setAPIVersion(result, i.group.APIVersion())
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

		setAPIVersion(result, i.group.APIVersion())
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

		setAPIVersion(result, i.group.APIVersion())
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

		setAPIVersion(result, i.group.APIVersion())
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

		params := PathParams(req)
		result, err := storage.DeleteCollection(ctx, deleteReq.IDs, &DeleteOptions{PathParams: params})
		if err != nil {
			handleError(i.serializer, err, w, req)
			return
		}

		setAPIVersion(result, i.group.APIVersion())
		WriteObjectNegotiated(i.serializer, w, req, http.StatusOK, result)
	}
}
