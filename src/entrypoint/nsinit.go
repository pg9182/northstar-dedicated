package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/kballard/go-shellquote"
)

func main() {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		fmt.Fprintf(os.Stderr, "Unsupported platform %s/%s.\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
		return
	}

	switch {
	default:
		fmt.Fprintf(os.Stderr, "This program is not intended to be run directly.\n")
		// Technically, it could be run directly outside a container, but first,
		// we'd need to allow the paths to be configured and initialize the
		// wineprefix on startup
		os.Exit(1)
		return

	case len(os.Args) == 2 && os.Args[1] == "__wineprefix__":
		if err := InitPrefix(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize and configure wineprefix.\n")
			os.Exit(1)
		}
		return

	case len(os.Args) == 2 && len(os.Args[1]) > 1 && strings.Trim(os.Args[1], " ") == "":
		break
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
	nso, err := MergeOverlay(
		"/tmp",
		"/mnt/titanfall",
		"/usr/lib/northstar",
		"/usr/lib/northstar-dedicated-d3d11",
		"/usr/lib/northstar-dedicated-gfsdk",
		"/mnt/mods",
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

	// This is needed since there's an issue with my current method of
	// eliminating x11 by simply making CreateWindow and ShowWindow always
	// successful in Wine's nulldrv. Apparently, it causes in-game time to slow
	// down for some users. More testing will be required.
	xvfb := 9

	fmt.Println("Starting Northstar...")
	sn, _ := nsc.Get("ns_server_name")
	updatetitle(sn)
	nsi, err := CreateInstance(nso, nsc, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v.\n", err)
		os.Exit(1)
		return
	}
	nsi.Output = os.Stdout
	nsi.Xvfb = &xvfb
	nsi.InfoCallback = func(ns NSInstanceStatus) {
		updatetitle(sn + " [" + ns.String() + "]")
	}
	fmt.Fprintf(os.Stderr, "Error: Failed to run server: %v.\n", nsi.Run())
	nsi.Close()
	os.Exit(1)
}

func updatetitle(t string) {
	// get access to the underlying data of the last arg
	h := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[1]))
	d := (*[1 << 30]byte)(unsafe.Pointer(h.Data))[:h.Len] // not including the null terminator

	// overwrite it
	for n := copy(d, []byte(t)); n < len(d); n++ {
		d[n] = 0 // fill the rest with nulls
	}
}
