package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// InitPrefix initializes the Wine prefix (at $WINEPREFIX) for Northstar.
func InitPrefix() error {
	fmt.Println("Initializing wineprefix...")
	{
		fmt.Println("wineboot")
		cmd := exec.Command("wineboot", "--init")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env(
			[]string{"PATH", "HOSTNAME", "HOME", "USER", "WINEPREFIX", "WINESERVER"},
			"WINEDLLOVERRIDES", "mscoree,mshtml=",
			"WINEARCH", "win64",
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("wineboot: %w", err)
		}
	}
	for _, reg := range []struct {
		Hive  string
		Key   string
		Entry string
		Type  string
		Data  string
	}{
		{`HKCU`, `Software\Wine`, `Version`, `REG_SZ`, `win10`},
		{`HKCU`, `Software\Wine\Drivers`, `Audio`, `REG_SZ`, ``},
		//{`HKCU`, `Software\Wine\Drivers`, `Graphics`, `REG_SZ`, `null`},
		{`HKCU`, `Software\Wine\WineDbg`, `ShowCrashDialog`, `REG_DWORD`, `0`},
		{`HKLM`, `System\CurrentControlSet\Services\WineBus`, `DisableHidraw`, `REG_DWORD`, `1`},
		{`HKLM`, `System\CurrentControlSet\Services\WineBus`, `DisableInput`, `REG_DWORD`, `1`},
		{`HKLM`, `System\CurrentControlSet\Services\WineBus`, `Enable SDL`, `REG_DWORD`, `0`},
		{`HKCU`, `Software\Wine\DllOverrides`, `d3d11`, `REG_SZ`, `native`},
		{`HKCU`, `Software\Wine\DllOverrides`, `mscoree`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `mshtml`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `wined3d`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `winevulkan`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `d3d9`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `d3d10`, `REG_SZ`, ``},
		{`HKCU`, `Software\Wine\DllOverrides`, `d3d12`, `REG_SZ`, ``},
	} {
		fmt.Printf("reg: [%s\\%s] %q = %s:%q\n", reg.Hive, reg.Key, reg.Entry, reg.Type, reg.Data)
		cmd := exec.Command("wine64", "reg", "add", reg.Hive+"\\"+reg.Key, "/v", reg.Entry, "/t", reg.Type, "/d", reg.Data)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env(
			[]string{"PATH", "HOSTNAME", "HOME", "USER", "WINEPREFIX", "WINESERVER"},
			"WINEDEBUG", "-all",
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("update registry: %w", err)
		}
	}
	if err := winewait(true, time.Second*30, false); err != nil {
		return fmt.Errorf("wine did not exit in time: %w", err)
	}
	fmt.Println("Done.")
	return nil
}

func winewait(verbose bool, timeout time.Duration, kill bool) error {
	ctx := context.Background()

	if timeout != 0 {
		x, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		ctx = x
	}

	if verbose {
		if kill {
			fmt.Printf("Killing Wine (limit: %s)...\n", timeout)
		} else {
			fmt.Printf("Waiting for Wine processes to exit (limit: %s)...\n", timeout)
		}
	}

	var cmd *exec.Cmd
	if kill {
		cmd = exec.CommandContext(ctx, "wineserver", "-k")
	} else {
		cmd = exec.CommandContext(ctx, "wineserver", "-w")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env(
		[]string{"PATH", "HOSTNAME", "HOME", "USER", "WINEPREFIX", "WINESERVER"},
		"WINEDEBUG", "-all",
	)

	if err := cmd.Run(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			if kill {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Failed to kill Wine.\n")
				}
			} else {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Wine did not exit in time. Killing.\n")
				}
				_ = winewait(false, time.Second, true)
			}
		} else {
			if verbose {
				if kill {
					fmt.Fprintf(os.Stderr, "Warning: Failed to kill Wine: %v.\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Failed to wait for Wine processes to exit: %v.\n", err)
				}
			}
		}
		return err
	}
	return nil
}
