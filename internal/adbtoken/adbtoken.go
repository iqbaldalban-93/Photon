package adbtoken

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"app/internal/tokenbinding"

	_ "modernc.org/sqlite"
)

const AliasPrefix = tokenbinding.ECDSAAliasPrefix

const (
	accountsCEDBPath = "/data/system_ce/0/accounts_ce.db"
	commandTimeout   = 30 * time.Second
)

var errADBRootUnavailable = errors.New("root access unavailable")

type Values map[string][]string

func ParseLoose(s string) Values {
	v := Values{}
	for _, pair := range strings.Split(s, "&") {
		if pair == "" {
			continue
		}
		k, val, _ := strings.Cut(pair, "=")
		v[k] = append(v[k], val)
	}
	return v
}

func (v Values) Get(key string) string {
	if len(v[key]) == 0 {
		return ""
	}
	return v[key][0]
}

func (v Values) Set(key, val string) {
	v[key] = []string{val}
}

func (v Values) Encode() string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, val := range v[k] {
			parts = append(parts, k+"="+val)
		}
	}
	return strings.Join(parts, "&")
}

func NeedsBinding(authString string) bool {
	params := ParseLoose(authString)
	if params.Get("token_binding_alias") != "" {
		return false
	}
	return params.Get("assertion_jwt") != "" ||
		params.Get("check_tb_upgrade_eligible") != ""
}

func AttachAlias(credentials, alias string) (string, error) {
	params := ParseLoose(credentials)
	params.Set("token_binding_alias", alias)
	return params.Encode(), nil
}

func ExtractAlias(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", errors.New("email cannot be empty")
	}
	if _, err := exec.LookPath("adb"); err != nil {
		return "", fmt.Errorf("adb was not found in PATH")
	}

	query := "select extras.value from extras join accounts on accounts._id=extras.accounts_id where accounts.name=? and extras.key='lstBindingKeyAlias';"

	devices, err := listADBDevices()
	if err != nil {
		return "", err
	}

	var failures []string
	var reachableRoot bool
	for _, device := range devices {
		rootCmd, rootCancel := adbCommand("-s", device, "root")
		_ = rootCmd.Run()
		rootCancel()

		alias, rooted, err := readTokenBindingAliasFromDevice(device, query, email)
		if rooted {
			reachableRoot = true
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", device, cleanADBError(err.Error())))
			continue
		}

		alias = strings.TrimSpace(alias)
		if alias == "" {
			failures = append(failures, fmt.Sprintf("%s: no token binding key for %s", device, email))
			continue
		}
		if !strings.HasPrefix(alias, AliasPrefix) {
			failures = append(failures, fmt.Sprintf("%s: unsupported token binding key format", device))
			continue
		}

		return alias, nil
	}

	if reachableRoot {
		return "", fmt.Errorf("token binding alias not found for %s on any connected adb device (%s)", email, strings.Join(failures, "; "))
	}
	return "", fmt.Errorf("could not read Android AccountManager on any connected adb device; root is required (%s)", strings.Join(failures, "; "))
}

func adbCommand(args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	cmd := exec.CommandContext(ctx, "adb", args...)
	cmd.WaitDelay = time.Second
	return cmd, cancel
}

func readTokenBindingAliasFromDevice(device, query, email string) (alias string, rooted bool, err error) {
	dbPath, cleanup, err := pullAccountsDB(device)
	if err != nil {
		return "", !errors.Is(err, errADBRootUnavailable), err
	}
	defer cleanup()

	alias, err = queryTokenBindingAlias(dbPath, query, email)
	if err != nil {
		return "", true, err
	}
	return alias, true, nil
}

func listADBDevices() ([]string, error) {
	cmd, cancel := adbCommand("devices")
	defer cancel()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list adb devices: %s", cleanADBError(string(out)))
	}

	var devices []string
	var unavailable []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] == "List" {
			continue
		}
		if fields[1] == "device" {
			devices = append(devices, fields[0])
		} else {
			unavailable = append(unavailable, fmt.Sprintf("%s is %s", fields[0], fields[1]))
		}
	}

	if len(devices) == 0 {
		if len(unavailable) > 0 {
			return nil, fmt.Errorf("no usable adb devices found (%s)", strings.Join(unavailable, "; "))
		}
		return nil, fmt.Errorf("no adb devices found")
	}

	return devices, nil
}

func pullAccountsDB(device string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "Photon-adb-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	mainDest := filepath.Join(tmpDir, "accounts_ce.db")
	if err := streamDeviceFile(device, accountsCEDBPath, mainDest); err != nil {
		cleanup()
		return "", nil, err
	}

	_ = streamDeviceFile(device, accountsCEDBPath+"-wal", mainDest+"-wal")
	_ = streamDeviceFile(device, accountsCEDBPath+"-shm", mainDest+"-shm")

	return mainDest, cleanup, nil
}

func streamDeviceFile(device, remotePath, localPath string) error {
	cmd, cancel := adbCommand("-s", device, "exec-out", "su", "-c", fmt.Sprintf("cat %q", remotePath))
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil || (stderr.Len() > 0 && stdout.Len() == 0) {
		msg := cleanADBError(stderr.String())
		if msg == "" && runErr != nil {
			msg = runErr.Error()
		}
		if isADBRootFailure(msg) {
			return fmt.Errorf("%w: %s", errADBRootUnavailable, msg)
		}
		return fmt.Errorf("failed to read %s: %s", remotePath, msg)
	}

	if err := os.WriteFile(localPath, stdout.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write local copy: %w", err)
	}
	return nil
}

func isADBRootFailure(msg string) bool {
	m := strings.ToLower(msg)
	if strings.Contains(m, "no such file") {
		return false
	}
	return strings.Contains(m, "su:") ||
		strings.Contains(m, "permission denied") ||
		strings.Contains(m, "not allowed") ||
		strings.Contains(m, "inaccessible or not found")
}

func queryTokenBindingAlias(dbPath, query, email string) (string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open accounts db: %w", err)
	}
	defer func() { _ = db.Close() }()

	var alias string
	if err := db.QueryRow(query, email).Scan(&alias); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to query token binding alias: %w", err)
	}
	return alias, nil
}

func cleanADBError(out string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return "adb command failed"
	}
	return strings.Join(strings.Fields(out), " ")
}
