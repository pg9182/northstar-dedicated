package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// NSInstance represents a ready-to-use instance of the Northstar dedicated
// server.
type NSInstance struct {
	Dir        string
	Executable string
	Args       []string

	Xvfb         *int
	Output       io.Writer
	InfoCallback InfoCallbackFunc

	mu         sync.Mutex    // for reading the pointers, which are initialized when Run is called and not set again (DO NOT HOLD THIS LOCK FOR A LONG TIME)
	terminated chan struct{} // close when terminating
	wait       chan struct{} // closed by run when the instance exits
	gameCmd    *exec.Cmd
}

// InfoCallbackFunc is function which will be called when the Northstar
// dedicated server status changes. It will not be called concurrently.
type InfoCallbackFunc func(NSInstanceStatus)

// NSInstanceStatus contains information about the state of a Northstar
// dedicated server.
type NSInstanceStatus struct {
	MapName      string
	PlaylistName string
	PlayerCount  int
	MaxPlayers   int
}

var titleRe = regexp.MustCompile(`^Titanfall 2 dedicated server - ([A-Za-z0-9_]+) ([0-9]+)/([0-9]+) players \(([A-Za-z0-9_]+)\)`) // TODO: updates: ensure this is still accurate

func (n *NSInstanceStatus) ParseTitle(title string) bool {
	m := titleRe.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	n.MapName = m[1]
	n.PlayerCount, _ = strconv.Atoi(m[2])
	n.MaxPlayers, _ = strconv.Atoi(m[3])
	n.PlaylistName = m[4]
	return true
}

// String formats the NSInstanceStatus.
func (n NSInstanceStatus) String() string {
	var sb strings.Builder
	if n.PlayerCount >= 0 {
		sb.WriteString(strconv.Itoa(n.PlayerCount))
	} else {
		sb.WriteByte('?')
	}
	if n.MaxPlayers >= 0 {
		sb.WriteByte('/')
		sb.WriteString(strconv.Itoa(n.MaxPlayers))
	} else {
		sb.WriteString("/?")
	}
	if n.MapName != "" {
		sb.WriteByte(' ')
		sb.WriteString(n.MapName)
	} else {
		sb.WriteString(" ???")
	}
	if n.PlaylistName != "" {
		sb.WriteByte(' ')
		sb.WriteString(n.PlaylistName)
	} else {
		sb.WriteString(" ???")
	}
	return sb.String()
}

// CreateInstance creates a new *NSInstance with the provided components.
func CreateInstance(overlay *NSOverlay, config *NSConfig, autoexec bool) (*NSInstance, error) {
	ins := &NSInstance{
		Dir:        overlay.Path,
		Executable: overlay.Executable(),
	}
	if autoexec {
		f, err := os.OpenFile(overlay.Autoexec(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("generate autoexec: %w", err)
		}
		if x, err := config.Autoexec(f); err == nil {
			ins.Args = x
		} else {
			return nil, fmt.Errorf("generate autoexec: %w", err)
		}
	} else {
		ins.Args = config.Arguments()
	}
	return ins, nil
}

// Run runs an instance and returns an error describing why the instance
// stopped. If there was not an error, this function will not return.
func (n *NSInstance) Run() error {
	if n.isUsed() {
		return fmt.Errorf("cannot re-use an instance")
	}

	n.mu.Lock()
	n.terminated = make(chan struct{})
	n.wait = make(chan struct{})
	n.mu.Unlock()
	defer close(n.wait)
	defer n.sendTerminate(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pty, err := setuppty()
	if err != nil {
		return fmt.Errorf("failed to allocate pty: %w", err)
	}
	defer pty.Close()

	titleCh := make(chan string, 1)
	go interceptsl(ctx, n.Output, pty.Master, titleCh)

	select {
	case <-n.terminated:
		return fmt.Errorf("server terminated")
	default:
	}

	xvfbResult := make(chan error, 1)
	gameResult := make(chan error, 1)

	n.mu.Lock()
	n.gameCmd = n.winecmd("wine64", append([]string{n.Executable, "-dedicated", "-multiple"}, n.Args...)...)
	n.gameCmd.Stdout = pty.Slave
	n.gameCmd.Stderr = pty.Slave
	n.gameCmd.Stdin = pty.Slave
	n.gameCmd.Env = append(n.gameCmd.Env,
		"WINEPATH="+n.Dir,
		"WINEDEBUG=fixme-secur32,fixme-bcrypt,fixme-ver,err-wldap32",
	)
	n.mu.Unlock()

	var xb bytes.Buffer
	if n.Xvfb != nil {
		display := ":" + strconv.Itoa(*n.Xvfb)

		if n.Output != nil {
			fmt.Fprintf(n.Output, "Starting xvfb on display %s...\n", display)
		}

		// clean up old xvfb lock file
		os.Remove("/tmp/.X" + strconv.Itoa(*n.Xvfb) + "-lock")

		xvfbCmd := exec.Command("Xvfb", display)
		xvfbCmd.Env = n.env()
		xvfbCmd.Stdout = &xb
		xvfbCmd.Stderr = &xb
		xvfbCmd.Stdin = nil

		if err := xvfbCmd.Start(); err != nil {
			return fmt.Errorf("failed to start xvfb: %w", err)
		}
		go func() {
			xvfbResult <- xvfbCmd.Wait()
		}()
		defer func() {
			if xvfbCmd.ProcessState != nil && !xvfbCmd.ProcessState.Exited() {
				xvfbCmd.Process.Signal(syscall.SIGKILL)
			}
		}()
		time.Sleep(time.Second * 2)

		n.gameCmd.Env = append(n.gameCmd.Env,
			"DISPLAY="+display,
		)
	}

	if err := n.gameCmd.Start(); err != nil {
		return fmt.Errorf("failed to start game (%q, %q, %q): %w", n.Dir, n.Executable, n.Args, err)
	}
	go func() {
		gameResult <- n.gameCmd.Wait()
	}()

	const (
		titleMatchFailCounterMax = 10
	)
	var (
		status                NSInstanceStatus
		titleMatchFailCounter int
		titleRateLimit        = ratelimit(ctx, time.Second/2, 1)
		watchdog              = CreateWatchdog(time.Minute*4, time.Minute)
	)
	for {
		select {
		case err := <-xvfbResult:
			if n.Output != nil {
				io.Copy(n.Output, &xb)
			}
			if err != nil {
				return fmt.Errorf("xvfb exited prematurely: %w", err)
			}
			return fmt.Errorf("xvfb exited prematurely")

		case err := <-gameResult: // waiter exited
			select {
			case <-n.terminated:
				return fmt.Errorf("server terminated")
			default:
				return fmt.Errorf("server exited: %w", err)
			}

		case <-ctx.Done(): // parent context is done
			return ctx.Err()

		case title := <-titleCh: // console title changed
			// note: this depends on a few things to work correctly:
			// - https://github.com/R2Northstar/NorthstarLauncher/blob/b10c8d6e4747a94ee93d7f6748c48319782dcc83/NorthstarDedicatedTest/dedicated.cpp#L57-L69
			//   updates every tick during game loop, title format
			// - https://github.com/wine-mirror/wine/blob/wine-7.0-rc3/programs/conhost/conhost.c#L2341-L2354
			//   conhost uses vt escape sequences
			// - https://github.com/wine-mirror/wine/blob/wine-7.0-rc3/dlls/kernelbase/console.c#L1745
			//   console output is passed through
			// - https://github.com/wine-mirror/wine/blob/wine-7.0-rc3/dlls/kernelbase/console.c#L1810-L1814
			//   goes through stdout if is_tty_handle
			// - https://github.com/wine-mirror/wine/blob/wine-7.0-rc3/dlls/kernelbase/console.c#L1791-L1794
			//   is_tty_handle impl
			// - https://github.com/wine-mirror/wine/blob/e909986e6ea5ecd49b2b847f321ad89b2ae4f6f1/dlls/ntdll/unix/env.c#L1176-L1181
			//   isatty

			if titleMatchFailCounter >= titleMatchFailCounterMax {
				break
			}

			select {
			case <-titleRateLimit:
				break
			default:
			}

			if !status.ParseTitle(title) {
				if titleMatchFailCounter++; titleMatchFailCounter == titleMatchFailCounterMax {
					fmt.Fprintf(os.Stderr, "warning: failed to parse status from game title %q: doesn't match %q\n", title, titleRe.String())
				}
				break
			}
			if n.InfoCallback != nil {
				n.InfoCallback(status)
			}
			watchdog.SendValidTitleUpdate()

		case <-watchdog.Event(): // watchdog event
			if err := watchdog.Err(); err != nil {
				switch {
				case errors.Is(err, ErrNSWatchdogUninitialized):
					fmt.Fprintf(os.Stderr, "warning: failed to initialize watchdog; engine error hangs will not be detected automatically: %v\n", err)
				case errors.Is(err, ErrNSWatchdogTimeout):
					fmt.Fprintf(os.Stderr, "watchdog: triggered (%v); killing server\n", err)
					return err
				}
			}
		}
	}
}

// Stop attempts to gracefully terminate the instance and waits for it to stop
// for up to the specified timeout.
func (n *NSInstance) Stop(timeout time.Duration) error {
	if !n.isUsed() {
		return fmt.Errorf("instance was never started")
	}

	t := time.NewTimer(timeout)
	defer t.Stop()

	n.sendTerminate(false)

	select {
	case <-t.C:
		return fmt.Errorf("instance did not stop within %s", timeout)
	case <-n.waitCh():
		return nil
	}
}

// Close forcefully terminates the instance. If it has already stopped, nothing
// is done. This function will not wait for the instance to exit, and err will
// always be nil unless the instance is not running.
func (n *NSInstance) Close() error {
	if !n.isUsed() {
		return fmt.Errorf("instance was never started")
	}
	n.sendTerminate(true)
	return nil
}

func (n *NSInstance) sendTerminate(force bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.terminated == nil {
		return
	}

	// close the channel if it is open
	select {
	case <-n.terminated:
	default:
		close(n.terminated)
	}

	select {
	case <-n.wait:
		return
	default:
	}

	if n.gameCmd != nil && n.gameCmd.ProcessState != nil && !n.gameCmd.ProcessState.Exited() {
		if force {
			n.gameCmd.Process.Signal(syscall.SIGKILL)
			n.winecmd("wineserver", "--kill").Start()
		} else {
			n.winecmd("wineboot", "--shutdown").Start()
		}
	}
}

func (n *NSInstance) isUsed() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.terminated != nil
}

func (n *NSInstance) waitCh() <-chan struct{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.wait
}

func (n *NSInstance) winecmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Env = n.env()
	cmd.Dir = n.Dir
	cmd.Stdout = n.Output
	cmd.Stderr = n.Output
	cmd.Stdin = nil
	return cmd
}

func (n *NSInstance) env(override ...string) []string {
	return env([]string{"PATH", "HOSTNAME", "HOME", "USER", "WINEPREFIX", "WINESERVER"},
		override...,
	)
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

// setuppty creates a PTY suitable for use with interceptsl.
func setuppty() (*PTY, error) {
	pty, err := OpenPTY()
	if err != nil {
		return nil, err
	}

	tos, err := pty.GetAttr()
	if err != nil {
		_ = pty.Close()
		return nil, err
	}

	// pty config
	tos.IFlag = BRKINT | IGNPAR | ISTRIP | IGNCR | IUTF8
	tos.OFlag = OPOST | ONOCR
	tos.CFlag = CREAD
	tos.LFlag = ISIG | ICANON

	// return from read() at least every 0.1s, whether or not data is available
	tos.CC[VMIN] = 0
	tos.CC[VTIME] = 1

	if err := pty.SetAttr(tos); err != nil {
		_ = pty.Close()
		return nil, err
	}

	if err := pty.SetWinsz(Winsize{
		Row:    25,
		Col:    120,
		XPixel: 0,
		YPixel: 0,
	}); err != nil {
		_ = pty.Close()
		return nil, err
	}

	return pty, nil
}

// interceptsl intercepts escape sequences to set the terminal title.
func interceptsl(ctx context.Context, w io.Writer, r io.Reader, title chan<- string) error {
	state := 0
	ibuf := make([]byte, 256)
	tbuf := make([]byte, 256)
	obuf := make([]byte, 256) // will use up to cap(ibuf) + cap(tbuf) + 4 in certain situations

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// note: Go will handle EAGAIN, EINTR, and polling internally while reading files
		// note: for a pty, VMIN=0 and VTIME=1 seems to work well (EAGAIN will be returned when nothing is available)
		// note: for writes, we don't need to check the count since Go io.Writers are supposed to always do full writes

		n, err := r.Read(ibuf)
		if err != nil {
			return err
		} else if n == 0 {
			continue
		}

		// fast path when no escape sequences in the buffer
		if state == 0 {
			for _, c := range ibuf[:n] {
				if c == 0x1B {
					goto slowpath
				}
			}
			if w != nil {
				_, _ = w.Write(ibuf[:n])
			}
			continue
		}

	slowpath:
		obuf = obuf[:0]
		for _, c := range ibuf[:n] {
			switch state {
			case 0: // normal output
				switch c {
				default:
					state = 0
					obuf = append(obuf, c)
				case 0x1B:
					state = 1
				}
			case 1: // at \x1B
				switch c {
				default:
					state = 0
					obuf = append(obuf, 0x1B, c)
				case ']':
					state = 2
				}
			case 2: // at \x1B]
				switch c {
				default:
					state = 0
					obuf = append(obuf, 0x1B, ']', c)
				case '0':
					state = 3
				}
			case 3: // at \x1B]0
				switch c {
				default:
					state = 0
					obuf = append(obuf, 0x1B, ']', '0', c)
				case ';':
					state = 4
					tbuf = tbuf[:0]
				}
			case 4: // in \x1B]0;
				switch c {
				default:
					// next title char
					if len(tbuf) < cap(tbuf) {
						tbuf = append(tbuf, c)
						break // out of the switch
					}
					fallthrough
				case 0x07:
					// end of title || overflow
					select {
					case title <- string(tbuf):
					default:
					}
					if len(tbuf) == cap(tbuf) {
						state = 5
					} else {
						state = 0
					}
					tbuf = tbuf[:0]
				case 0x1B:
					// start of a new escape sequence (this shouldn't happen)
					state = 1
				}
			case 5: // in title
				switch c {
				default:
					// overflowing title character
				case 0x07:
					// end of the overflowing title
					state = 0
				case 0x1B:
					// start of a new escape sequence (this shouldn't happen)
					state = 1
				}
			}
		}
		if w != nil {
			_, _ = w.Write(obuf)
		}
	}
}

// ratelimit creates a chan to be used for rate limiting. It is destroyed when
// ctx ends.
func ratelimit(ctx context.Context, interval time.Duration, burst int) <-chan struct{} {
	ch := make(chan struct{}, burst)
	go func() {
		tk := time.NewTicker(interval)
		for {
			select {
			case <-tk.C:
				select {
				case ch <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				close(ch)
				tk.Stop()
				return
			}
		}
	}()
	return ch
}

// NSWatchdog detects engine errors resulting in server hangs.
//
// This works because Northstar updates the title every tick (i.e. multiple
// times per second), and engine errors suspend the game thread. Thus, once we
// start receiving valid title updates quickly, we can start a watchdog timer,
// then reset it on every title update.
type NSWatchdog struct {
	initTarget uint32
	initCtr    uint32       // atomic
	last       atomic.Value // time.Time
	init       time.Duration
	interval   time.Duration
	timer      *time.Timer
}

// Errors for NSWatchdog (compare with errors.Is).
var (
	ErrNSWatchdogUninitialized = errors.New("watchdog did not receive enough title updates for initialization")
	ErrNSWatchdogTimeout       = errors.New("watchdog did not receive a title update in time")
)

// CreateWatchdog initializes a new watchdog and starts the initial timer.
func CreateWatchdog(initTimeout time.Duration, interval time.Duration) *NSWatchdog {
	n := &NSWatchdog{
		initTarget: 10, // require 10 title events within the interval for a successful init
		initCtr:    0,
		init:       initTimeout,
		interval:   interval,
		timer:      time.NewTimer(initTimeout),
	}
	n.last.Store(time.Now())
	return n
}

// IsInitialized checks if the initial tick target has been reached.
func (n *NSWatchdog) IsInitialized() bool {
	return atomic.LoadUint32(&n.initCtr) >= n.initTarget
}

// Err returns an error if the watchdog is currently returning.
func (n *NSWatchdog) Err() error {
	if n.IsInitialized() {
		if d := time.Now().Sub(n.last.Load().(time.Time)); d >= n.interval {
			return fmt.Errorf("%w: last tick was %s ago", ErrNSWatchdogTimeout, d)
		}
	} else {
		if d := time.Now().Sub(n.last.Load().(time.Time)); d >= n.init {
			return fmt.Errorf("%w: only received %d/%d initial ticks less than %s apart within %s", ErrNSWatchdogUninitialized, atomic.LoadUint32(&n.initCtr), n.initTarget, n.interval, n.init)
		}
	}
	return nil
}

// Event returns a channel which sends the current time at which the watchdog is
// triggered. Check Err for the current status. Note: the channel is buffered.
func (n *NSWatchdog) Event() <-chan time.Time {
	if n.timer == nil {
		return nil // will block forever when receiving
	}
	return n.timer.C
}

// SendValidTitleUpdate updates the watchdog.
func (n *NSWatchdog) SendValidTitleUpdate() {
	if n.IsInitialized() {
		n.last.Store(time.Now())
		n.timer.Reset(n.interval)
	} else {
		t, p := time.Now(), n.last.Load().(time.Time)
		n.last.Store(t)

		if t.Sub(p) > n.interval {
			atomic.StoreUint32(&n.initCtr, 0)
		} else {
			if atomic.AddUint32(&n.initCtr, 1) == n.initTarget {
				n.timer.Reset(n.interval)
			}
		}
	}
}

// Stop cleans up resources allocated by the watchdog.
func (n *NSWatchdog) Stop() {
	n.timer.Stop()
}
