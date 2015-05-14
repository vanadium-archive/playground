// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Handlers for HTTP requests to save and load playground examples.
//
// handlerSave() handles a POST request with bundled playground source code.
// The bundle is persisted in a database and a unique ID returned.
// handlerLoad() handles a GET request with an id parameter. It returns the
// bundle saved under the provided ID or slug, if any.
// handlerListDefault() handles a GET request with no parameters. It returns
// a list of descriptions of all default bundles. Default bundles are saved
// using the pgadmin tool, not the HTTP API.
// The current implementation uses a MySQL-like SQL database for persistence.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"v.io/x/playground/lib/log"
	"v.io/x/playground/lib/storage"
)

//////////////////////////////////////////
// HTTP request handlers

// GET request that returns the saved bundle for the given ID or slug.
func handlerLoad(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method and read GET parameters.
	if !checkGetMethod(w, r) {
		return
	}
	bIdOrSlug := r.FormValue("id")
	if bIdOrSlug == "" {
		storageError(w, http.StatusBadRequest, "Must specify id to load.")
		return
	}

	bLink, bData, err := storage.GetBundleByLinkIdOrSlug(bIdOrSlug)
	if err == storage.ErrNotFound {
		storageError(w, http.StatusNotFound, "No data found for provided id.")
		return
	} else if err != nil {
		storageInternalError(w, "Error getting bundleLink for id/slug ", bIdOrSlug, ": ", err)
		return
	}

	storageRespond(w, http.StatusOK, fullResponseFromLinkAndData(bLink, bData))
}

// POST request that saves the body as a new bundle and returns the bundle id.
func handlerSave(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method and read POST body.
	// Limit is set to maxSize+1 to allow distinguishing between exactly maxSize
	// and larger than maxSize requests.
	requestBody := getPostBody(w, r, *maxSize+1)
	if requestBody == nil {
		return
	}
	if len(requestBody) > *maxSize {
		storageError(w, http.StatusBadRequest, "Program too large.")
		return
	}

	// TODO(ivanpi): Check if bundle is parseable. Format/lint?

	bLink, bData, err := storage.StoreBundleLinkAndData(string(requestBody))
	if err != nil {
		storageInternalError(w, "Error storing bundle: ", err)
		return
	}

	storageRespond(w, http.StatusOK, fullResponseFromLinkAndData(bLink, bData))
}

// GET request that returns a list of default bundle descriptions.
func handlerListDefault(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method. No GET parameters are currently used by /list.
	if !checkGetMethod(w, r) {
		return
	}

	bList, err := storage.GetDefaultBundleList()
	if err != nil {
		storageInternalError(w, "Error getting default bundle list: ", err)
		return
	}

	bListResp := make([]*BundleDescResponse, 0, len(bList))
	for _, bLink := range bList {
		bListResp = append(bListResp, descResponseFromLink(bLink))
	}

	storageRespond(w, http.StatusOK, bListResp)
}

//////////////////////////////////////////
// Response handling

type ErrorResponse struct {
	Error string `json:"error"`
}

type BundleDescResponse struct {
	// Bundle ID of the saved/loaded bundle.
	Link string `json:"link"`
	// Slug of the saved/loaded bundle.
	// Currently set only for most recent versions of default bundles.
	Slug string `json:"slug,omitempty"`
	// Creation timestamp of the loaded bundle.
	// Since the timestamp is set by the database, /save responses omit it.
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

type BundleFullResponse struct {
	// Bundle description, as sent in /list response.
	BundleDescResponse
	// Contents of the saved/loaded bundle.
	Data string `json:"data"`
}

func descResponseFromLink(bLink *storage.BundleLink) *BundleDescResponse {
	return &BundleDescResponse{
		Link:      bLink.Id,
		Slug:      string(bLink.Slug),
		CreatedAt: zeroTimeToNil(bLink.CreatedAt),
	}
}

func fullResponseFromLinkAndData(bLink *storage.BundleLink, bData *storage.BundleData) *BundleFullResponse {
	return &BundleFullResponse{
		BundleDescResponse: *descResponseFromLink(bLink),
		Data:               bData.Json,
	}
}

// Converts time to pointer, mapping zero time to nil to force it to be
// omitted from JSON. See https://github.com/golang/go/issues/5218
func zeroTimeToNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// Sends response to client. Request handler should exit after this call.
func storageRespond(w http.ResponseWriter, status int, body interface{}) {
	bodyJson, _ := json.Marshal(body)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(bodyJson)))
	w.WriteHeader(status)
	w.Write(bodyJson)
}

// Sends error response with specified message to client.
func storageError(w http.ResponseWriter, status int, msg string) {
	storageRespond(w, status, &ErrorResponse{
		Error: msg,
	})
}

// Logs error internally and sends non-specific error response to client.
func storageInternalError(w http.ResponseWriter, v ...interface{}) {
	if len(v) > 0 {
		log.Error(v...)
	}
	storageError(w, http.StatusInternalServerError, "Internal error, please retry.")
}
