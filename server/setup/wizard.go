package setup

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// clientModes returns all supported client version strings.
func clientModes() []string {
	return []string{
		"S1.0", "S1.5", "S2.0", "S2.5", "S3.0", "S3.5", "S4.0", "S5.0", "S5.5", "S6.0", "S7.0",
		"S8.0", "S8.5", "S9.0", "S10", "FW.1", "FW.2", "FW.3", "FW.4", "FW.5", "G1", "G2", "G3",
		"G3.1", "G3.2", "GG", "G5", "G5.1", "G5.2", "G6", "G6.1", "G7", "G8", "G8.1", "G9", "G9.1",
		"G10", "G10.1", "Z1", "Z2", "ZZ",
	}
}

// FinishRequest holds the user's configuration choices from the wizard.
type FinishRequest struct {
	DBHost            string `json:"dbHost"`
	DBPort            int    `json:"dbPort"`
	DBUser            string `json:"dbUser"`
	DBPassword        string `json:"dbPassword"`
	DBName            string `json:"dbName"`
	Host              string `json:"host"`
	ClientMode        string `json:"clientMode"`
	AutoCreateAccount bool   `json:"autoCreateAccount"`
}

// buildDefaultConfig produces a config map matching config.example.json structure
// with the user's values merged in.
func buildDefaultConfig(req FinishRequest) map[string]interface{} {
	config := map[string]interface{}{
		"Host":                   req.Host,
		"BinPath":                "bin",
		"Language":               "en",
		"DisableSoftCrash":       false,
		"HideLoginNotice":        true,
		"LoginNotices":           []string{"<BODY><CENTER><SIZE_3><C_4>Welcome to Erupe!"},
		"PatchServerManifest":    "",
		"PatchServerFile":        "",
		"DeleteOnSaveCorruption": false,
		"ClientMode":             req.ClientMode,
		"QuestCacheExpiry":       300,
		"CommandPrefix":          "!",
		"AutoCreateAccount":      req.AutoCreateAccount,
		"LoopDelay":              50,
		"DefaultCourses":         []int{1, 23, 24},
		"EarthStatus":            0,
		"EarthID":                0,
		"EarthMonsters":          []int{0, 0, 0, 0},
		"Screenshots": map[string]interface{}{
			"Enabled":       true,
			"Host":          "127.0.0.1",
			"Port":          8080,
			"OutputDir":     "screenshots",
			"UploadQuality": 100,
		},
		"SaveDumps": map[string]interface{}{
			"Enabled":    true,
			"RawEnabled": false,
			"OutputDir":  "save-backups",
		},
		"Capture": map[string]interface{}{
			"Enabled":         false,
			"OutputDir":       "captures",
			"ExcludeOpcodes":  []int{},
			"CaptureSign":     true,
			"CaptureEntrance": true,
			"CaptureChannel":  true,
		},
		"DebugOptions": map[string]interface{}{
			"CleanDB":             false,
			"MaxLauncherHR":       false,
			"LogInboundMessages":  false,
			"LogOutboundMessages": false,
			"LogMessageData":      false,
			"MaxHexdumpLength":    256,
			"DivaOverride":        0,
			"FestaOverride":       -1,
			"TournamentOverride":  0,
			"DisableTokenCheck":   false,
			"QuestTools":          false,
			"AutoQuestBackport":   true,
			"ProxyPort":           0,
			"CapLink": map[string]interface{}{
				"Values": []int{51728, 20000, 51729, 1, 20000},
				"Key":    "",
				"Host":   "",
				"Port":   80,
			},
		},
		"GameplayOptions": map[string]interface{}{
			"MinFeatureWeapons":              0,
			"MaxFeatureWeapons":              1,
			"MaximumNP":                      100000,
			"MaximumRP":                      50000,
			"MaximumFP":                      120000,
			"TreasureHuntExpiry":             604800,
			"DisableLoginBoost":              false,
			"DisableBoostTime":               false,
			"BoostTimeDuration":              7200,
			"ClanMealDuration":               3600,
			"ClanMemberLimits":               [][]int{{0, 30}, {3, 40}, {7, 50}, {10, 60}},
			"BonusQuestAllowance":            3,
			"DailyQuestAllowance":            1,
			"LowLatencyRaviente":             false,
			"RegularRavienteMaxPlayers":      8,
			"ViolentRavienteMaxPlayers":      8,
			"BerserkRavienteMaxPlayers":      32,
			"ExtremeRavienteMaxPlayers":      32,
			"SmallBerserkRavienteMaxPlayers": 8,
			"GUrgentRate":                    0.10,
			"GCPMultiplier":                  1.00,
			"HRPMultiplier":                  1.00,
			"HRPMultiplierNC":                1.00,
			"SRPMultiplier":                  1.00,
			"SRPMultiplierNC":                1.00,
			"GRPMultiplier":                  1.00,
			"GRPMultiplierNC":                1.00,
			"GSRPMultiplier":                 1.00,
			"GSRPMultiplierNC":               1.00,
			"ZennyMultiplier":                1.00,
			"ZennyMultiplierNC":              1.00,
			"GZennyMultiplier":               1.00,
			"GZennyMultiplierNC":             1.00,
			"MaterialMultiplier":             1.00,
			"MaterialMultiplierNC":           1.00,
			"GMaterialMultiplier":            1.00,
			"GMaterialMultiplierNC":          1.00,
			"ExtraCarves":                    0,
			"ExtraCarvesNC":                  0,
			"GExtraCarves":                   0,
			"GExtraCarvesNC":                 0,
			"DisableHunterNavi":              false,
			"MezFesSoloTickets":              5,
			"MezFesGroupTickets":             1,
			"MezFesDuration":                 172800,
			"MezFesSwitchMinigame":           false,
			"EnableKaijiEvent":               false,
			"EnableHiganjimaEvent":           false,
			"EnableNierEvent":                false,
			"DisableRoad":                    false,
			"SeasonOverride":                 false,
		},
		"Discord": map[string]interface{}{
			"Enabled":  false,
			"BotToken": "",
			"RelayChannel": map[string]interface{}{
				"Enabled":          false,
				"MaxMessageLength": 183,
				"RelayChannelID":   "",
			},
		},
		"Commands": []map[string]interface{}{
			{"Name": "Help", "Enabled": true, "Description": "Show enabled chat commands", "Prefix": "help"},
			{"Name": "Rights", "Enabled": false, "Description": "Overwrite the Rights value on your account", "Prefix": "rights"},
			{"Name": "Raviente", "Enabled": true, "Description": "Various Raviente siege commands", "Prefix": "ravi"},
			{"Name": "Teleport", "Enabled": false, "Description": "Teleport to specified coordinates", "Prefix": "tele"},
			{"Name": "Reload", "Enabled": true, "Description": "Reload all players in your Land", "Prefix": "reload"},
			{"Name": "KeyQuest", "Enabled": false, "Description": "Overwrite your HR Key Quest progress", "Prefix": "kqf"},
			{"Name": "Course", "Enabled": true, "Description": "Toggle Courses on your account", "Prefix": "course"},
			{"Name": "PSN", "Enabled": true, "Description": "Link a PlayStation Network ID to your account", "Prefix": "psn"},
			{"Name": "Discord", "Enabled": true, "Description": "Generate a token to link your Discord account", "Prefix": "discord"},
			{"Name": "Ban", "Enabled": false, "Description": "Ban/Temp Ban a user", "Prefix": "ban"},
			{"Name": "Timer", "Enabled": true, "Description": "Toggle the Quest timer", "Prefix": "timer"},
			{"Name": "Playtime", "Enabled": true, "Description": "Show your playtime", "Prefix": "playtime"},
		},
		"Courses": []map[string]interface{}{
			{"Name": "HunterLife", "Enabled": true},
			{"Name": "Extra", "Enabled": true},
			{"Name": "Premium", "Enabled": true},
			{"Name": "Assist", "Enabled": false},
			{"Name": "N", "Enabled": false},
			{"Name": "Hiden", "Enabled": false},
			{"Name": "HunterSupport", "Enabled": false},
			{"Name": "NBoost", "Enabled": false},
			{"Name": "NetCafe", "Enabled": true},
			{"Name": "HLRenewing", "Enabled": true},
			{"Name": "EXRenewing", "Enabled": true},
		},
		"Database": map[string]interface{}{
			"Host":     req.DBHost,
			"Port":     req.DBPort,
			"User":     req.DBUser,
			"Password": req.DBPassword,
			"Database": req.DBName,
		},
		"Sign": map[string]interface{}{
			"Enabled": true,
			"Port":    53312,
		},
		"API": map[string]interface{}{
			"Enabled":     true,
			"Port":        8080,
			"PatchServer": "",
			"Banners":     []interface{}{},
			"Messages":    []interface{}{},
			"Links":       []interface{}{},
			"LandingPage": map[string]interface{}{
				"Enabled": true,
				"Title":   "My Frontier Server",
				"Content": "<p>Welcome! Server is running.</p>",
			},
		},
		"Channel": map[string]interface{}{
			"Enabled": true,
		},
		"Entrance": map[string]interface{}{
			"Enabled": true,
			"Port":    53310,
			"Entries": []map[string]interface{}{
				{
					"Name": "Newbie", "Description": "", "IP": "", "Type": 3, "Recommended": 2, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54001, "MaxPlayers": 100, "Enabled": true},
						{"Port": 54002, "MaxPlayers": 100, "Enabled": true},
					},
				},
				{
					"Name": "Normal", "Description": "", "IP": "", "Type": 1, "Recommended": 0, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54003, "MaxPlayers": 100, "Enabled": true},
						{"Port": 54004, "MaxPlayers": 100, "Enabled": true},
					},
				},
				{
					"Name": "Cities", "Description": "", "IP": "", "Type": 2, "Recommended": 0, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54005, "MaxPlayers": 100, "Enabled": true},
					},
				},
				{
					"Name": "Tavern", "Description": "", "IP": "", "Type": 4, "Recommended": 0, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54006, "MaxPlayers": 100, "Enabled": true},
					},
				},
				{
					"Name": "Return", "Description": "", "IP": "", "Type": 5, "Recommended": 0, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54007, "MaxPlayers": 100, "Enabled": true},
					},
				},
				{
					"Name": "MezFes", "Description": "", "IP": "", "Type": 6, "Recommended": 6, "AllowedClientFlags": 0,
					"Channels": []map[string]interface{}{
						{"Port": 54008, "MaxPlayers": 100, "Enabled": true},
					},
				},
			},
		},
	}

	return config
}

// writeConfig writes the config map to config.json with pretty formatting.
func writeConfig(config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile("config.json", data, 0600); err != nil {
		return fmt.Errorf("writing config.json: %w", err)
	}
	return nil
}

// detectOutboundIP returns the preferred outbound IPv4 address.
func detectOutboundIP() (string, error) {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("detecting outbound IP: %w", err)
	}
	defer func() { _ = conn.Close() }()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.To4().String(), nil
}

// testDBConnection tests connectivity to the PostgreSQL server and checks
// whether the target database and its tables exist.
func testDBConnection(host string, port int, user, password, dbName string) (*DBStatus, error) {
	status := &DBStatus{}

	// Connect to the 'postgres' maintenance DB to check if target DB exists.
	adminConn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' password='%s' dbname='postgres' sslmode=disable",
		host, port, user, password,
	)
	adminDB, err := sql.Open("postgres", adminConn)
	if err != nil {
		return nil, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}
	defer func() { _ = adminDB.Close() }()

	if err := adminDB.Ping(); err != nil {
		return nil, fmt.Errorf("cannot reach PostgreSQL: %w", err)
	}
	status.ServerReachable = true

	var exists bool
	err = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return status, fmt.Errorf("checking database existence: %w", err)
	}
	status.DatabaseExists = exists

	if !exists {
		return status, nil
	}

	// Connect to the target DB to check for tables.
	targetConn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode=disable",
		host, port, user, password, dbName,
	)
	targetDB, err := sql.Open("postgres", targetConn)
	if err != nil {
		return status, nil
	}
	defer func() { _ = targetDB.Close() }()

	var tableCount int
	err = targetDB.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		return status, nil
	}
	status.TablesExist = tableCount > 0
	status.TableCount = tableCount

	return status, nil
}

// DBStatus holds the result of a database connectivity check.
type DBStatus struct {
	ServerReachable bool `json:"serverReachable"`
	DatabaseExists  bool `json:"databaseExists"`
	TablesExist     bool `json:"tablesExist"`
	TableCount      int  `json:"tableCount"`
}

// createDatabase creates the target database by connecting to the 'postgres' maintenance DB.
func createDatabase(host string, port int, user, password, dbName string) error {
	adminConn := fmt.Sprintf(
		"host='%s' port='%d' user='%s' password='%s' dbname='postgres' sslmode=disable",
		host, port, user, password,
	)
	db, err := sql.Open("postgres", adminConn)
	if err != nil {
		return fmt.Errorf("connecting to PostgreSQL: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Database names can't be parameterized; validate it's alphanumeric + underscores.
	for _, c := range dbName {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return fmt.Errorf("invalid database name %q: only alphanumeric characters and underscores allowed", dbName)
		}
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	return nil
}

// applyInitSchema runs pg_restore to load the init.sql (PostgreSQL custom dump format).
func applyInitSchema(host string, port int, user, password, dbName string) error {
	pgRestore, err := exec.LookPath("pg_restore")
	if err != nil {
		return fmt.Errorf("pg_restore not found in PATH: %w (install PostgreSQL client tools)", err)
	}

	schemaPath := filepath.Join("schemas", "init.sql")
	if _, err := os.Stat(schemaPath); err != nil {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}

	cmd := exec.Command(pgRestore,
		"--host", host,
		"--port", fmt.Sprint(port),
		"--username", user,
		"--dbname", dbName,
		"--no-owner",
		"--no-privileges",
		schemaPath,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w\n%s", err, string(output))
	}
	return nil
}

// collectSQLFiles returns sorted .sql filenames from a directory.
func collectSQLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// applySQLFiles executes all .sql files in a directory in sorted order.
func applySQLFiles(db *sql.DB, dir string) ([]string, error) {
	files, err := collectSQLFiles(dir)
	if err != nil {
		return nil, err
	}

	var applied []string
	for _, f := range files {
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			return applied, fmt.Errorf("reading %s: %w", f, err)
		}
		_, err = db.Exec(string(data))
		if err != nil {
			return applied, fmt.Errorf("executing %s: %w", f, err)
		}
		applied = append(applied, f)
	}
	return applied, nil
}
