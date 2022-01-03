package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// Termios IOCTLs.
const (
	TCGETS     = 0x5401     // get terminal attributes
	TCSETS     = 0x5402     // set terminal attributes
	TIOCGWINSZ = 0x5413     // get the terminal window size
	TIOCSWINSZ = 0x5414     // set the terminal window size
	TIOCGPTN   = 0x80045430 // get the PTY number
	TIOCSPTLCK = 0x40045431 // lock/unlock PTY
)

// Termios input flags.
const (
	IGNBRK  = 0000001 // ignore BREAK condition
	BRKINT  = 0000002 // map BREAK to SIGINTR
	IGNPAR  = 0000004 // ignore (discard) parity errors
	PARMRK  = 0000010 // mark parity and framing errors
	INPCK   = 0000020 // enable checking of parity errors
	ISTRIP  = 0000040 // strip 8th bit off chars
	INLCR   = 0000100 // map NL into CR
	IGNCR   = 0000200 // ignore CR
	ICRNL   = 0000400 // map CR to NL (ala CRMOD)
	IUCLC   = 0001000
	IXON    = 0002000 // enable output flow control
	IXANY   = 0004000 // any char will restart after stop
	IXOFF   = 0010000 // enable input flow control
	IMAXBEL = 0020000 // ring bell on input queue full
	IUTF8   = 0040000 // maintain state for UTF-8 VERASE
)

// Termios output flags.
const (
	OPOST  = 0000001 // enable following output processing
	OLCUC  = 0000002
	ONLCR  = 0000004 // map NL to CR-NL (ala CRMOD)
	OCRNL  = 0000010 // map CR to NL on output
	ONOCR  = 0000020 // no CR output at column 0
	ONLRET = 0000040 // NL performs CR function
	OFILL  = 0000100 // use fill characters for delay
	OFDEL  = 0000200 // fill is DEL, else NUL
	NLDLY  = 0000400 // \n delay
	NL0    = 0000000
	NL1    = 0000400
	CRDLY  = 0003000 // \r delay
	CR0    = 0000000
	CR1    = 0001000
	CR2    = 0002000
	CR3    = 0003000
	TABDLY = 0014000 // horizontal tab delay
	TAB0   = 0000000
	TAB1   = 0004000
	TAB2   = 0010000
	TAB3   = 0014000
	BSDLY  = 0020000 // \b delay
	BS0    = 0000000
	BS1    = 0020000
	FFDLY  = 0100000 // form feed delay
	FF0    = 0000000
	FF1    = 0100000
	VTDLY  = 0040000 // vertical tab delay
	VT0    = 0000000
	VT1    = 0040000
	XTABS  = 0014000
)

// Termios control modes.
const (
	CSIZE  = 0000060 // used as mask when setting character size
	CS5    = 0000000 // char size 5bits
	CS6    = 0000020 // char size 6bits
	CS7    = 0000040 // char size 7bits
	CS8    = 0000060 // char size 8bits
	CSTOPB = 0000100 // two stop bits
	CREAD  = 0000200 // enable input
	PARENB = 0000400 // generate and expect parity bit
	PARODD = 0001000 // set odd parity
	HUPCL  = 0002000 // send HUP when last process closes term
	CLOCAL = 0004000 // no modem control signals
)

// Termios modes.
const (
	ISIG    = 0000001 // enable signals INTR, QUIT, [D]SUSP
	ICANON  = 0000002 // canonicalize input lines
	XCASE   = 0000004
	ECHO    = 0000010 // enable echoing
	ECHOE   = 0000020 // visually erase chars
	ECHOK   = 0000040 // echo NL after line kill
	ECHONL  = 0000100 // echo NL even if ECHO is off
	NOFLSH  = 0000200 // don't flush after interrupt
	TOSTOP  = 0000400 // stop background jobs from output
	ECHOCTL = 0001000 // echo control chars as ^(Char)
	ECHOPRT = 0002000 // visual erase mode for hardcopy
	ECHOKE  = 0004000 // visual erase for line kill
	FLUSHO  = 0010000 // output being flushed (state)
	PENDIN  = 0040000 // XXX retype pending input (state)
	IEXTEN  = 0100000 // enable DISCARD and LNEXT
	EXTPROC = 0200000 // external processing
)

// Termios control characters.
const (
	VINTR    = 0  // char will send an interrupt signal
	VQUIT    = 1  // char will send a quit signal
	VERASE   = 2  // char will erase last typed char
	VKILL    = 3  // char will erase current line
	VEOF     = 4  // char will send EOF
	VTIME    = 5  // set read timeout in tenths of seconds
	VMIN     = 6  // set min characters for a complete read
	VSWTC    = 7  // char will switch to a different shell layer
	VSTART   = 8  // char will restart output after stopping it
	VSTOP    = 9  // char will stop output
	VSUSP    = 10 // char will send a stop signal
	VEOL     = 11 // char will end the line
	VREPRINT = 12 // will redraw the current line
	VDISCARD = 13
	VWERASE  = 14 // char will erase last word typed
	VLNEXT   = 15 // char will enter the next char quoted
	VEOL2    = 16 // char alternate to end line
	NCCS     = 32 // Termios CC size
)

type Termios struct {
	IFlag  uint32     // input modes
	OFlag  uint32     // output modes
	CFlag  uint32     // control modes
	LFlag  uint32     // local modes
	CC     [NCCS]byte // control characters
	ISpeed uint32     // input speed
	OSpeed uint32     // output speed
}

type Winsize struct {
	Row    uint16 // number of rows
	Col    uint16 // number of columns
	XPixel uint16 // width in pixels
	YPixel uint16 // height in pixels
}

type PTY struct {
	Master *os.File
	Slave  *os.File
}

func OpenPTY() (*PTY, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var unlock int
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(master.Fd()), uintptr(TIOCSPTLCK), uintptr(unsafe.Pointer(&unlock))); err != 0 {
		_ = master.Close()
		return nil, err
	}

	var n int
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(master.Fd()), uintptr(TIOCGPTN), uintptr(unsafe.Pointer(&n))); err != 0 {
		_ = master.Close()
		return nil, err
	}

	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, err
	}

	return &PTY{master, slave}, nil
}

func (p *PTY) Close() error {
	var err1, err2 error
	if p.Slave != nil {
		err1 = p.Slave.Close()
	}
	if p.Master != nil {
		err2 = p.Master.Close()
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (p *PTY) GetAttr() (Termios, error) {
	var t Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.Slave.Fd()), uintptr(TCGETS), uintptr(unsafe.Pointer(&t))); errno != 0 {
		return t, fmt.Errorf("get pty attributes: %w", errno)
	}
	return t, nil
}

func (p *PTY) SetAttr(t Termios) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.Slave.Fd()), uintptr(TCSETS), uintptr(unsafe.Pointer(&t))); errno != 0 {
		return fmt.Errorf("set pty attributes: %w", errno)
	}
	return nil
}

func (p *PTY) SetWinsz(w Winsize) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(p.Slave.Fd()), uintptr(TIOCSWINSZ), uintptr(unsafe.Pointer(&w))); errno != 0 {
		return fmt.Errorf("set pty window size: %w", errno)
	}
	return nil
}
