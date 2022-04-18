package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/kballard/go-shellquote"
)

func main() {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		fmt.Fprintf(os.Stderr, "Unsupported platform %s/%s.\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		if v := os.Getenv("HOSTNAME"); v != "" {
			hostname = v
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Couldn't get container hostname: %v.\n", err)
		}
	}
	if hostname == "" {
		hostname = "unknown"
	}
	for _, v := range []string{"NS_SERVER_NAME", "NS_SERVER_DESC"} {
		os.Setenv(v, strings.ReplaceAll(os.Getenv(v), "{{hostname}}", hostname))
	}

	fmt.Println("Northstar Dedicated Server - Docker (hostname: " + hostname + ")")
	fmt.Println()
	fmt.Println("    https://northstar.tf")
	fmt.Println("    https://northstar.tf/discord")
	fmt.Println("    https://github.com/R2Northstar/Northstar")
	fmt.Println("    https://github.com/pg9182/northstar-dedicated")
	fmt.Println()

	fmt.Println("Merging files...")
	if pts, err := mounts("/mnt", "/usr"); err == nil {
		fmt.Println()
		fmt.Println("    Bind mounts:")
		var unsupported bool
		for _, pt := range pts {
			fmt.Printf("        %s\n", pt)
			if strings.HasPrefix(pt, "/usr") {
				unsupported = true
			}
		}
		if unsupported {
			fmt.Println()
			fmt.Println("    Warning: You have overridden internal files. This may break without warning in future versions.")
		}
	}
	nso, err := MergeOverlay(
		"/tmp",
		"/mnt/titanfall",
		"/usr/lib/northstar",
		"/mnt/mods",
		"/mnt/navs",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to merge game files: %v.\n", err)
		os.Exit(1)
		return
	}
	defer nso.Delete()
	fmt.Println()

	fmt.Println("Merging configuration...")
	port := "37015"
	if v, ok := os.LookupEnv("NS_PORT"); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid port %q.\n", v)
			os.Exit(1)
			return
		} else if n < 1 || n > 65535 {
			fmt.Fprintf(os.Stderr, "Error: Invalid port %q: out of range.\n", v)
			os.Exit(1)
			return
		} else {
			port = v
		}
	}
	as, err := shellquote.Split(os.Getenv("NS_EXTRA_ARGUMENTS"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to split extra arguments %#q: %v.\n", os.Getenv("NS_EXTRA_ARGUMENTS"), err)
		os.Exit(1)
		return
	}
	nsc := MergeConfig(
		map[string]string{
			// from R2Northstar/mods/Northstar.CustomServers/mod/cfg/autoexec_ns_server.cfg @ v1.3.0
			"ns_server_name":                      "",
			"ns_server_desc":                      "",
			"ns_server_password":                  "",
			"ns_report_server_to_masterserver":    "1",
			"ns_report_sp_server_to_masterserver": "0",
			"ns_auth_allow_insecure":              "0",
			"ns_erase_auth_info":                  "1",
			"ns_player_auth_port":                 "8081",
			"ns_masterserver_hostname":            "https://northstar.tf",
			"everything_unlocked":                 "1",
			"ns_should_return_to_lobby":           "1",
			"net_chan_limit_mode":                 "2",
			"net_chan_limit_msec_per_sec":         "100",
			"sv_querylimit_per_sec":               "15",
			"base_tickinterval_mp":                "0.016666667",
			"sv_updaterate_mp":                    "20",
			"sv_minupdaterate":                    "20",
			"sv_max_snapshots_multiplayer":        "300",
			"net_data_block_enabled":              "0",
			"host_skip_client_dll_crc":            "1",
		},
		map[string]string{
			// only include commonly-used ones here
			"ns_player_auth_port":                 "NS_PORT_AUTH",
			"ns_server_name":                      "NS_SERVER_NAME",
			"ns_server_desc":                      "NS_SERVER_DESC",
			"ns_server_password":                  "NS_SERVER_PASSWORD",
			"ns_masterserver_hostname":            "NS_MASTERSERVER_URL",
			"ns_report_server_to_masterserver":    "NS_MASTERSERVER_REGISTER",
			"ns_report_sp_server_to_masterserver": "NS_MASTERSERVER_REGISTER",
			"ns_auth_allow_insecure":              "NS_INSECURE",
		},
		append([]string{
			"-port", port,
		}, as...)...,
	)
	if cerrs := nsc.Validate(); cerrs != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to merge config:\n")
		for c, errs := range cerrs {
			v, _ := nsc.Get(c)
			for _, err := range errs {
				fmt.Fprintf(os.Stderr, "           %s (%q) - %v\n", c, v, err)
			}
		}
		os.Exit(2)
		return
	}
	fmt.Println()
	nsc.Display(os.Stdout, "    ")
	fmt.Println()

	var buf bytes.Buffer
	args, err := nsc.Autoexec(&buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: generate autoexec: %v.\n", err)
		os.Exit(1)
		return
	}
	if err := os.WriteFile(nso.Autoexec(), buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: write autoexec: %v.\n", err)
		os.Exit(1)
		return
	}

	fmt.Println("Starting Northstar...")

	sn, _ := nsc.Get("ns_server_name")

	cmd := &exec.Cmd{
		Path: "/usr/bin/nswrap",
		Args: append(append([]string{"nswrap", nso.Path}, args...)),
		Env: env([]string{"PATH", "HOSTNAME", "HOME", "USER", "WINEPREFIX", "WINESERVER"},
			"NSWRAP_TITLE", sn,
			"DISPLAY", "xvfb",
		),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range ch {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Run()

	if err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			os.Exit(ex.ExitCode())
			return
		}
		fmt.Fprintf(os.Stderr, "Error: Failed to run northstar: %v.\n", err)
		os.Exit(1)
	}
}

func env(preserve []string, override ...string) []string {
	var r []string
	if len(override)%2 != 0 {
		panic("invalid env override")
	}
	for _, x := range os.Environ() {
		spl := strings.SplitN(x, "=", 2)
		for i := 0; i < len(override); i += 2 {
			if override[i] == spl[0] {
				continue
			}
		}
		for _, p := range preserve {
			if spl[0] == p {
				r = append(r, x)
				break
			}
		}
	}
	for i := 0; i < len(override); i += 2 {
		r = append(r, override[i]+"="+override[i+1])
	}
	return r
}

func mounts(base ...string) ([]string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	var r []string
	for s.Scan() {
		v := strings.Fields(s.Text())
		if len(v) < 4 {
			continue
		}
		for _, x := range base {
			if strings.HasPrefix(v[1], x) {
				r = append(r, v[1]+" "+v[2]+" "+v[3])
				continue
			}
		}
	}
	return r, s.Err()
}
