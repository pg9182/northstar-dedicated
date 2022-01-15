package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

type NSConfig struct {
	cv map[string]string
	cs map[string]string
	ce map[string]string
	ax []string
}

func MergeConfig(defaults, env map[string]string, args ...string) *NSConfig {
	var n NSConfig
	n.ApplyValues(defaults)
	n.ApplyEnv(env, os.LookupEnv)
	n.ApplyArgs(args...)
	return &n
}

func (n *NSConfig) Set(c, v, s string) {
	if n.cv == nil {
		n.cv = map[string]string{}
	}
	if n.cs == nil {
		n.cs = map[string]string{}
	}
	if n.ce == nil {
		n.ce = map[string]string{}
	}
	n.cv[c] = v
	n.cs[c] = s
}

func (n *NSConfig) ApplyValues(cv map[string]string) {
	for c, v := range cv {
		n.Set(c, v, "default")
	}
}

func (n *NSConfig) ApplyEnv(ce map[string]string, getenv func(string) (string, bool)) {
	for c, e := range ce {
		if v, ok := getenv(e); ok {
			n.Set(c, v, "env:"+e)
		}
		n.ce[c] = e
	}
}

func (n *NSConfig) ApplyArgs(ax ...string) {
	var x, c, v string
	for len(ax) != 0 {
		// shift and get the arg
		x, ax = ax[0], ax[1:]
		if x == "" || x[0] != '+' {
			n.ax = append(n.ax, x)
			continue
		}

		// shift and get the convar value
		c, v, ax = x[1:], ax[0], ax[1:]
		if _, ok := n.cv[c]; ok {
			n.Set(c, v, "arg")
		} else {
			n.ax = append(n.ax, x, v)
		}
	}
}

// Autoexec writes the config as an autoexec file and returns the remaining
// command-line arguments.
func (n *NSConfig) Autoexec(w io.Writer) ([]string, error) {
	cx := make([]string, 0, len(n.cv))
	for c := range n.cv {
		cx = append(cx, c)
	}
	sort.Strings(cx)
	for _, c := range cx {
		if _, err := fmt.Fprintf(w, "%s \"%s\"\r\n", c, n.cv[c]); err != nil {
			return nil, err
		}
	}
	return n.ax, nil
}

// Arguments gets the config as a list of command-line arguments.
//
// Note: As of Northstar v1.3.0, for some reason, the ns_masterserver_hostname
// convar completely breaks if passed this way (it'll give obscure errors when
// trying to register).
func (n *NSConfig) Arguments(order ...string) []string {
	if n.cv == nil {
		return nil
	}

	// start with the other arguments
	args := make([]string, len(n.ax))
	copy(args, n.ax)

	// append the ordered convars if they're set
	seen := make(map[string]struct{})
	for _, c := range order {
		if v, ok := n.cv[c]; ok {
			args = append(args, "+"+c, v)
		}
		seen[c] = struct{}{}
	}

	// sort the remaining convars
	cx := make([]string, 0, len(n.cv))
	for c := range n.cv {
		if _, skip := seen[c]; !skip {
			cx = append(cx, c)
		}
	}
	sort.Strings(cx)

	// append the remaing convars
	for _, c := range cx {
		args = append(args, "+"+c, n.cv[c])
	}

	return args
}

func (n *NSConfig) Get(convar string) (string, bool) {
	if n.cv == nil {
		return "", false
	}
	v, ok := n.cv[convar]
	return v, ok
}

func (n *NSConfig) Display(w io.Writer, prefix string) error {
	if n.cv == nil || n.cs == nil || n.ce == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%sExtra arguments:\n", prefix); err != nil {
		return err
	}
	for _, a := range n.ax {
		if _, err := fmt.Fprintf(w, "%s    %s\n", prefix, a); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%sConfig:\n", prefix); err != nil {
		return err
	}
	cx := make([]string, 0, len(n.cv))
	for c := range n.cv {
		cx = append(cx, c)
	}
	sort.Strings(cx)
	for _, c := range cx {
		var e string
		if x, ok := n.ce[c]; ok {
			e = " (" + x + ")"
		}
		if _, err := fmt.Fprintf(w, "%s    +%s %q (from %s)%s\n", prefix, c, n.cv[c], n.cs[c], e); err != nil {
			return err
		}
	}
	return nil
}

func (n *NSConfig) Validate() map[string][]error {
	if n.cv == nil {
		return nil
	}
	errs := make(map[string][]error)

	if n.cv["ns_server_name"] == "" {
		errs["ns_server_name"] = append(errs["ns_server_name"], fmt.Errorf("server name is required"))
	}

	cx := make([]string, 0, len(n.cv))
	for c := range n.cv {
		cx = append(cx, c)
	}
	sort.Strings(cx)

	for _, c := range cx {
		v := n.cv[c]
		if strings.ContainsAny(v, "\"") {
			errs[c] = append(errs[c], fmt.Errorf("invalid value: must not contain double quotes"))
		}
		switch c {
		case "ns_server_name", "ns_server_desc", "ns_server_password":
			// string
		case "ns_masterserver_hostname":
			if u, err := url.Parse(v); err != nil {
				errs[c] = append(errs[c], fmt.Errorf("invalid url: %v", err))
			} else if u.Scheme != "http" && u.Scheme != "https" {
				errs[c] = append(errs[c], fmt.Errorf("invalid url: scheme must be http or https"))
			} else if u.Host == "" {
				errs[c] = append(errs[c], fmt.Errorf("invalid url: hostname must be set"))
			}
		case "ns_report_server_to_masterserver", "ns_report_sp_server_to_masterserver", "ns_auth_allow_insecure", "ns_erase_auth_info", "everything_unlocked", "ns_should_return_to_lobby", "net_data_block_enabled", "host_skip_client_dll_crc":
			if v != "0" && v != "1" {
				errs[c] = append(errs[c], fmt.Errorf("invalid bool (0 or 1)"))
			}
		case "ns_player_auth_port", "net_chan_limit_mode", "net_chan_limit_msec_per_sec", "sv_querylimit_per_sec", "sv_updaterate_mp", "sv_minupdaterate", "sv_max_snapshots_multiplayer":
			if _, err := strconv.ParseInt(v, 10, 64); err != nil {
				errs[c] = append(errs[c], fmt.Errorf("invalid integer: %v", err))
			}
		case "base_tickinterval_mp":
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				errs[c] = append(errs[c], fmt.Errorf("invalid float: %v", err))
			}
		}
	}

	if len(errs) == 0 {
		errs = nil
	}
	return errs
}
