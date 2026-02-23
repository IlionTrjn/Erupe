package setup

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

//go:embed wizard.html
var wizardHTML embed.FS

// wizardServer holds state for the setup wizard HTTP handlers.
type wizardServer struct {
	logger *zap.Logger
	done   chan struct{} // closed when setup is complete
}

func (ws *wizardServer) handleIndex(w http.ResponseWriter, _ *http.Request) {
	data, err := wizardHTML.ReadFile("wizard.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (ws *wizardServer) handleDetectIP(w http.ResponseWriter, _ *http.Request) {
	ip, err := detectOutboundIP()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ip": ip})
}

func (ws *wizardServer) handleClientModes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"modes": clientModes()})
}

// testDBRequest is the JSON body for POST /api/setup/test-db.
type testDBRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
}

func (ws *wizardServer) handleTestDB(w http.ResponseWriter, r *http.Request) {
	var req testDBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	status, err := testDBConnection(req.Host, req.Port, req.User, req.Password, req.DBName)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"error":  err.Error(),
			"status": status,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": status})
}

// initDBRequest is the JSON body for POST /api/setup/init-db.
type initDBRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	DBName     string `json:"dbName"`
	CreateDB   bool   `json:"createDB"`
	ApplyInit  bool   `json:"applyInit"`
	ApplyUpdate bool  `json:"applyUpdate"`
	ApplyPatch bool   `json:"applyPatch"`
	ApplyBundled bool `json:"applyBundled"`
}

func (ws *wizardServer) handleInitDB(w http.ResponseWriter, r *http.Request) {
	var req initDBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	var log []string
	addLog := func(msg string) {
		log = append(log, msg)
		ws.logger.Info(msg)
	}

	if req.CreateDB {
		addLog(fmt.Sprintf("Creating database '%s'...", req.DBName))
		if err := createDatabase(req.Host, req.Port, req.User, req.Password, req.DBName); err != nil {
			addLog(fmt.Sprintf("ERROR: %s", err))
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
			return
		}
		addLog("Database created successfully")
	}

	if req.ApplyInit {
		addLog("Applying init schema (pg_restore)...")
		if err := applyInitSchema(req.Host, req.Port, req.User, req.Password, req.DBName); err != nil {
			addLog(fmt.Sprintf("ERROR: %s", err))
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
			return
		}
		addLog("Init schema applied successfully")
	}

	// For update/patch/bundled schemas, connect to the target DB.
	if req.ApplyUpdate || req.ApplyPatch || req.ApplyBundled {
		connStr := fmt.Sprintf(
			"host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode=disable",
			req.Host, req.Port, req.User, req.Password, req.DBName,
		)
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			addLog(fmt.Sprintf("ERROR connecting to database: %s", err))
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
			return
		}
		defer func() { _ = db.Close() }()

		applyDir := func(dir, label string) bool {
			addLog(fmt.Sprintf("Applying %s schemas from %s...", label, dir))
			applied, err := applySQLFiles(db, filepath.Join("schemas", dir))
			for _, f := range applied {
				addLog(fmt.Sprintf("  Applied: %s", f))
			}
			if err != nil {
				addLog(fmt.Sprintf("ERROR: %s", err))
				return false
			}
			addLog(fmt.Sprintf("%s schemas applied (%d files)", label, len(applied)))
			return true
		}

		if req.ApplyUpdate {
			if !applyDir("update-schema", "update") {
				writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
				return
			}
		}
		if req.ApplyPatch {
			if !applyDir("patch-schema", "patch") {
				writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
				return
			}
		}
		if req.ApplyBundled {
			if !applyDir("bundled-schema", "bundled") {
				writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "log": log})
				return
			}
		}
	}

	addLog("Database initialization complete!")
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "log": log})
}

func (ws *wizardServer) handleFinish(w http.ResponseWriter, r *http.Request) {
	var req FinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	config := buildDefaultConfig(req)
	if err := writeConfig(config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ws.logger.Info("config.json written successfully")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	// Signal completion — this will cause the HTTP server to shut down.
	close(ws.done)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
