package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/kardianos/service"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
	"github.com/oneworkspacestudio/my-vault/internal/vault"
)

var version = "development"

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func defaultData() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	var marker struct {
		Data string `json:"data"`
	}
	if platform.LoadJSON(filepath.Join(dir, "myvault-location.json"), &marker) == nil && filepath.IsAbs(marker.Data) {
		return marker.Data
	}
	if platform.LoadJSON(filepath.Join(dir, "workspace-location.json"), &marker) == nil && filepath.IsAbs(marker.Data) {
		return filepath.Join(marker.Data, "my-vault")
	}
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "OneWorkspace", "MyVault")
}
func run() error {
	action := "run"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action = args[0]
		args = args[1:]
	}
	f := flag.NewFlagSet(action, flag.ContinueOnError)
	data := f.String("data", defaultData(), "Private local NTFS data folder")
	listen := f.String("listen", "", "Fixed loopback listener")
	origin := f.String("origin", "", "Dedicated HTTPS URL, or local loopback HTTP")
	noBrowser := f.Bool("no-browser", false, "Do not open local owner tools")
	input := f.String("input", "", "Independent age-encrypted backup")
	output := f.String("output", "", "New .age backup file")
	stdin := f.Bool("request-stdin", false, "Read backup secret as JSON on standard input; never command-line arguments")
	if e := f.Parse(args); e != nil {
		return e
	}
	if action == "version" {
		fmt.Println(version)
		return nil
	}
	root, e := filepath.Abs(*data)
	if e != nil || filepath.Dir(root) == root {
		return errors.New("choose a dedicated absolute data folder")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	switch action {
	case "trust-local":
		certificate := filepath.Join(root, "local-tls", "localhost.cer")
		if e = platform.NoReparse(certificate); e != nil {
			return e
		}
		return platform.TrustLocalCertificate(certificate)
	case "manage":
		return vault.Control(root, "manage")
	case "stop":
		if e = vault.Control(root, "stop"); e != nil {
			return e
		}
		return vault.WaitStopped(root)
	case "restore", "backup", "verify":
		if !*stdin {
			if action == "restore" {
				return vault.RestoreWizard(ctx, root, *input, *listen, *origin)
			}
			return vault.Control(root, "manage")
		}
		var request struct {
			Password  string `json:"password"`
			Identity  string `json:"identity"`
			Recipient string `json:"recipient"`
		}
		if e = json.NewDecoder(io.LimitReader(os.Stdin, 1<<20)).Decode(&request); e != nil {
			return errors.New("provide a JSON request through standard input")
		}
		if action == "restore" {
			return vault.Restore(root, *input, request.Password, request.Identity, *listen, *origin)
		}
		lock, e := platform.Lock(filepath.Join(root, "runtime.lock"))
		if e != nil {
			return errors.New("stop My Vault first, or use the local management window for a coordinated backup")
		}
		defer lock.Close()
		if action == "backup" {
			record, e := vault.BackupStopped(root, *output, request.Password, request.Recipient)
			if e != nil {
				return e
			}
			return json.NewEncoder(os.Stdout).Encode(record)
		}
		stage, e := vault.VerifyBackup(root, *input, request.Password, request.Identity)
		if e != nil {
			return e
		}
		defer os.RemoveAll(stage)
		fmt.Println("Every file digest and SQLite integrity check passed. Account unlock is a separate recovery check.")
		return nil
	case "run":
		c, e := vault.Load(root, *listen, *origin)
		if e != nil {
			return e
		}
		if !service.Interactive() {
			return runService(root, c)
		}
		return vault.Run(ctx, root, c, !*noBrowser)
	case "service-install", "service-start", "service-stop", "service-uninstall":
		return controlService(root, action)
	default:
		return errors.New("use run, manage, stop, backup, verify, restore, version or service-install/start/stop/uninstall")
	}
}
