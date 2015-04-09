// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Handlers for HTTP requests to save and load playground examples.
//
// handlerSave() handles a POST request with bundled playground source code.
// The bundle is persisted in a database and a unique ID returned.
// handlerLoad() handles a GET request with an id parameter. It returns the
// bundle saved under the provided ID, if any.
// The current implementation uses a MySQL-like SQL database for persistence.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"playground/compilerd/storage"
)

//////////////////////////////////////////
// HTTP request handlers

// GET request that returns the saved bundle for the given id.
func handlerLoad(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method and read GET parameters.
	if !checkGetMethod(w, r) {
		return
	}
	bId := r.FormValue("id")
	if bId == "" {
		storageError(w, http.StatusBadRequest, "Must specify id to load.")
		return
	}
	bData, err := storage.GetBundleDataByLinkId(bId)
	if err == storage.ErrNotFound {
		storageError(w, http.StatusNotFound, "No data found for provided id.")
		return
	} else if err != nil {
		storageInternalError(w, "Error getting bundleLink for id", bId, ":", err)
		return
	}

	storageRespond(w, http.StatusOK, &StorageResponse{
		Link: bId,
		Data: bData.Json,
	})
	return
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

	bLink, bData, err := storage.StoreBundleLinkAndData(requestBody)
	if err != nil {
		storageInternalError(w, err)
		return
	}

	storageRespond(w, http.StatusOK, &StorageResponse{
		Link: bLink.Id,
		Data: bData.Json,
	})
}

//////////////////////////////////////////
// Response handling

type StorageResponse struct {
	// Error message. If empty, request was successful.
	Error string
	// Bundle ID for the saved/loaded bundle.
	Link string
	// Contents of the loaded bundle.
	Data string
}

// Sends response to client. Request handler should exit after this call.
func storageRespond(w http.ResponseWriter, status int, body *StorageResponse) {
	bodyJson, _ := json.Marshal(body)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(bodyJson)))
	w.WriteHeader(status)
	w.Write(bodyJson)
}

// Sends error response with specified message to client.
func storageError(w http.ResponseWriter, status int, msg string) {
	storageRespond(w, status, &StorageResponse{
		Error: msg,
	})
}

// Logs error internally and sends non-specific error response to client.
func storageInternalError(w http.ResponseWriter, v ...interface{}) {
	if len(v) > 0 {
		log.Println(v...)
	}
	storageError(w, http.StatusInternalServerError, "Internal error, please retry.")
}
