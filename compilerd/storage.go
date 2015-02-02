// Handlers for HTTP requests to save and load playground examples.
//
// handlerSave() handles a POST request with bundled playground source code.
// The bundle is persisted in a database and a unique ID returned.
// handlerLoad() handles a GET request with an id parameter. It returns the
// bundle saved under the provided ID, if any.
// The current implementation uses a MySQL-like SQL database for persistence.

package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"v.io/playground/lib"
	"v.io/playground/lib/lsql"
)

var (
	// Database connection string as specified in
	// https://github.com/go-sql-driver/mysql/#dsn-data-source-name
	// Query parameters are not supported.
	sqlConf = flag.String("sqlconf", "", "The go-sql-driver database connection string. If empty, load and save requests are disabled.")

	// Testing parameter, use default value for production.
	// Name of dataset to use. Used as table name prefix (a single SQL database
	// can contain multiple datasets).
	dataset = flag.String("dataset", "pg", "Testing: Name of dataset to use (and create if needed, when allowed by setupdb).")

	// If set, will attempt to create any missing database tables for the given
	// dataset. Idempotent.
	setupDB = flag.Bool("setupdb", false, "Whether to create missing database tables for dataset.")
)

//////////////////////////////////////////
// SqlData type definitions

// === High-level schema ===
// Each bundle save request generates a unique BundleID.
// Every BundleID corresponds to exactly one bundle Json, stored as a Unicode
// text blob. However, a single bundle Json can correspond to an unlimited
// number of BundleIDs.
// BundleIDs are mapped to Jsons via the BundleHash, which is a hash value of
// the Json. The bundleLink table stores records of BundleID (primary key) and
// BundleHash. The bundleData table stores records of BundleHash (primary key)
// and Json. The bundleLink.BundleHash column references bundleData.BundleHash,
// but can be set to NULL to allow deletion of bundleData entries.
// The additional layer of indirection allows storing identical bundles more
// efficiently and makes the bundle ID independent of its contents, allowing
// implementation of change history, sharing, expiration etc.
// TODO(ivanpi): Revisit ON DELETE SET NULL when delete functionality is added.
// === Schema type details ===
// The BundleID is a 64-character string consisting of URL-friendly characters
// (alphanumeric, '-', '_', '.', '~'), beginning with an underscore.
// The BundleHash is a raw (sequence of 32 bytes) SHA256 hash.
// The Json is a MEDIUMTEXT (up to 16 MiB) Unicode (utf8mb4) blob.
// Note: If bundles larger than ~1 MiB are to be stored, the max_allowed_packed
// SQL connection parameter must be increased.
// TODO(ivanpi): Normalize the Json (e.g. file order).

// All SqlData types should be added here, in order of table initialization.
var dataTypes = []lsql.SqlData{&bundleData{}, &bundleLink{}}

type bundleData struct {
	// Raw SHA256 of the bundle contents
	BundleHash []byte // primary key
	// The bundle contents
	Json string
}

func (bd *bundleData) TableName() string {
	return *dataset + "_bundleData"
}

func (bd *bundleData) TableDef() *lsql.SqlTable {
	return lsql.NewSqlTable(bd, "BundleHash", []lsql.SqlColumn{
		{Name: "BundleHash", Type: "BINARY(32)", Null: false},
		{Name: "Json", Type: "MEDIUMTEXT", Null: false},
	}, []string{})
}

func (bd *bundleData) QueryRefs() []interface{} {
	return []interface{}{&bd.BundleHash, &bd.Json}
}

type bundleLink struct {
	// 64-byte printable ASCII string
	BundleID string // primary key
	// Raw SHA256 of the bundle contents
	BundleHash []byte // foreign key => bundleData.BundleHash
	// TODO(ivanpi): Add creation (and expiration, last access?) timestamps.
}

func (bl *bundleLink) TableName() string {
	return *dataset + "_bundleLink"
}

func (bl *bundleLink) TableDef() *lsql.SqlTable {
	return lsql.NewSqlTable(bl, "BundleID", []lsql.SqlColumn{
		{Name: "BundleID", Type: "CHAR(64) CHARACTER SET ascii", Null: false},
		{Name: "BundleHash", Type: "BINARY(32)", Null: true},
	}, []string{
		"FOREIGN KEY (BundleHash) REFERENCES " + (&bundleData{}).TableName() + "(BundleHash) ON DELETE SET NULL",
	})
}

func (bl *bundleLink) QueryRefs() []interface{} {
	return []interface{}{&bl.BundleID, &bl.BundleHash}
}

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

	if !checkDBInit(w) {
		return
	}

	var bLink bundleLink
	// Get the entry for the provided id.
	err := dbhRead.QFetch(bId, &bLink)
	if err == lsql.ErrNoSuchEntity {
		storageError(w, http.StatusNotFound, "No data found for provided id.")
		return
	} else if err != nil {
		storageInternalError(w, "Error getting bundleLink for id", bId, ":", err)
		return
	}

	var bData bundleData
	// Get the bundle data for the hash linked in the entry.
	// Note: This can fail if the bundle is deleted between fetching bundleLink
	// and bundleData. However, it is highly unlikely, costly to mitigate (using
	// a serializable transaction), and unimportant (error 500 instead of 404).
	err = dbhRead.QFetch(bLink.BundleHash, &bData)
	if err != nil {
		storageInternalError(w, "Error getting bundleData for id", bId, ":", err)
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
	requestBody := getPostBody(w, r)
	if requestBody == nil {
		return
	}
	if len(requestBody) > maxSize {
		storageError(w, http.StatusBadRequest, "Program too large.")
		return
	}

	if !checkDBInit(w) {
		return
	}

	// TODO(ivanpi): Check if bundle is parseable. Format/lint?

	bHashFixed := rawHash(requestBody)
	bHash := bHashFixed[:]

	randomLink := func() string {
		h := make([]byte, 16, 16+len(bHash))
		err := binary.Read(crand.Reader, binary.LittleEndian, h)
		if err != nil {
			panic(fmt.Errorf("rng failed: %v", err))
		}
		return "_" + stringHash(append(h, bHash...))[1:]
	}

	bNewData := bundleData{
		BundleHash: bHash,
		Json:       string(requestBody),
	}
	bNewLink := bundleLink{
		BundleID:   randomLink(),
		BundleHash: bHash,
	}

	// TODO(ivanpi): The function is not idempotent, there is a small probability
	// of making a duplicate entry (if a commit succeeds, but a transient network
	// issue reports it as failed).
	err := dbhSeq.RunInTransaction(3, func(txh *lsql.DBHandle) error {
		// If a bundleLink entry exists for the generated BundleID, regenerate
		// BundleID and retry. Buying lottery ticket optional.
		bLinkFound, err := txh.QExists(bNewLink.BundleID, &bNewLink)
		if err != nil {
			log.Println("error checking bundleLink existence for id", bNewLink.BundleID, ":", err)
			return err
		} else if bLinkFound {
			log.Println("random generation resulted in duplicate id", bNewLink.BundleID)
			bNewLink.BundleID = randomLink()
			return lsql.RetryTransaction
		}

		// Check if a bundleData entry exists for this BundleHash.
		bDataFound, err := txh.QExists(bNewData.BundleHash, &bNewData)
		if err != nil {
			log.Println("error checking bundleData existence for hash", hex.EncodeToString(bHash), ":", err)
			return err
		} else if !bDataFound {
			// If not, save the bundleData.
			err = txh.QInsert(&bNewData)
			if err != nil {
				log.Println("error storing bundleData for hash", hex.EncodeToString(bHash), ":", err)
				return err
			}
		}

		// Save the bundleLink with the generated BundleID referring to the
		// bundleData.
		err = txh.QInsert(&bNewLink)
		if err != nil {
			log.Println("error storing bundleLink for id", bNewLink.BundleID, ":", err)
			return err
		}

		return nil
	})

	if err == nil {
		storageRespond(w, http.StatusOK, &StorageResponse{
			Link: bNewLink.BundleID,
			Data: bNewData.Json,
		})
	} else if err == lsql.ErrTooManyRetries {
		storageInternalError(w, err)
	} else {
		// An informative error message has already been printed.
		storageInternalError(w)
	}
	return
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

//////////////////////////////////////////
// SQL database handles

// Data writes for the schema are complex enough to require transactions with
// SERIALIZABLE isolation. However, reads do not require SERIALIZABLE. Since
// database/sql only allows setting transaction isolation per connection,
// a separate connection with only READ-COMMITTED isolation is used for reads
// to reduce lock contention and deadlock frequency.

var (
	// Database handle with SERIALIZABLE transaction isolation.
	// Used for read-write transactions.
	dbhSeq *lsql.DBHandle
	// Database handle with READ_COMMITTED transaction isolation.
	// Used for non-transactional reads.
	dbhRead *lsql.DBHandle
)

func newDBHandle(sqlConfig, txIsolation string, dataTypes []lsql.SqlData, setupdb, readonly bool) (*lsql.DBHandle, error) {
	// Create a database handle.
	dbc, err := lib.NewDBConn(sqlConfig, txIsolation)
	if err != nil {
		return nil, err
	}
	dbh := lsql.NewDBHandle(dbc)
	if setupdb {
		// Create missing database tables.
		for _, t := range dataTypes {
			if err := dbh.CreateTable(t, true, lib.CreateTableSuffix); err != nil {
				return nil, fmt.Errorf("failed initializing database tables: %v", err)
			}
			// TODO(ivanpi): Initialize database with fixed-ID examples?
		}
	}
	// Prepare simple database queries.
	for _, t := range dataTypes {
		if err := dbh.RegisterType(t, readonly); err != nil {
			return nil, fmt.Errorf("failed preparing database queries: %v", err)
		}
	}
	return dbh, nil
}

func initDBHandles() error {
	if *sqlConf == "" {
		return nil
	}

	var err error
	// If setupDB is set, tables should be initialized only once, on the handle
	// that is opened first.
	if dbhSeq, err = newDBHandle(*sqlConf, "SERIALIZABLE", dataTypes, *setupDB, false); err != nil {
		return err
	}
	// The READ-COMMITTED connection is used only for reads, so it is not
	// necessary to prepare writing statements such as INSERT.
	if dbhRead, err = newDBHandle(*sqlConf, "READ-COMMITTED", dataTypes, false, true); err != nil {
		return err
	}

	return nil
}

func checkDBInit(w http.ResponseWriter) bool {
	if dbhSeq == nil || dbhRead == nil {
		storageError(w, http.StatusInternalServerError, "Internal error: cannot connect to database.")
		return false
	}

	return true
}
