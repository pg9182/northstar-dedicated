/**
 * Linux wrapper for the Northstar dedicated server.
 *
 * - Requires Linux 3.4+, but 5.3+ is recommended.
 * - The caller must manage the WINEPREFIX; only wine/NorthstarLauncher/Xvfb is managed.
 * - Tested on vanilla Wine 7.0, but pg9182's patched Wine build should be used if possible.
 * - Currently requires X11, even though no window is shown. If the stubs aren't used, a full X11
 *   server and graphics is required.
 * - Requires an assembled Northstar v1.4.0+ game dir, preferably with pg9182's d3d11 and gfsdk stubs.
 */

#define _GNU_SOURCE
#define _POSIX_C_SOURCE 200809L

#ifndef __linux__
#error "nswrap only supports linux"
#endif

#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <regex.h>
#include <poll.h>
#include <sched.h>
#include <signal.h>
#include <stdatomic.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <termios.h>
#include <unistd.h>

#include <sys/epoll.h>
#include <sys/ioctl.h>
#include <sys/prctl.h>
#include <sys/timerfd.h>
#include <sys/types.h>
#include <sys/signalfd.h>
#include <sys/sysinfo.h>
#include <sys/utsname.h>
#include <sys/wait.h>

/** The default value for WINEDEBUG if it is not provided (to clean up irrelevant log spam). */
#define WINEDEBUG_DEFAULT "fixme-secur32,fixme-bcrypt,fixme-ver,err-wldap32"

/** The number of logical cores required to run the Northstar dedicated server. */
#define NS_REQUIRED_CORES 3

/** The chunk size for console i/o (also the maximum length of a parsed title). */
#define NS_IOPROC_OUTPUT_CHUNK_SIZE 256

/** The regexp for matching the console title against to extract the server status. */
#define NS_STATUS_RE(_x, _int, _str) _x( \
    " - ([A-Za-z0-9_]+) ([0-9]+)/([0-9]+) players \\(([A-Za-z0-9_]+)\\)", \
    _str(map_name) _int(player_count) _int(max_players) _str(playlist_name) \
)

/** Log functions. All output is prefixed and written to stderr. */
#define ns_log(fmt, ...) fprintf(stderr, "nswrap: " fmt "\n", ##__VA_ARGS__)
#define ns_perror(fmt, ...) ns_log(fmt ": %m", ##__VA_ARGS__)
#define ns_perror_dbg(fmt, ...) ns_perror("debug: error: " fmt " (in %s) (%s:%d)", ##__VA_ARGS__, __FUNCTION__, __FILE__, __LINE__)

/** Executes a block while preserving errno. */
#define preserve_errno(x) do { \
    int errno_saved = errno; \
    {x} \
    errno = errno_saved; \
} while(0)

/** Hacky defer mechanism using GCC nested funcs and the cleanup attribute. */
void __defer_cb(void (**fn)(void)) { (*fn)(); }
#define __defer_paste(x, y) x ## y
#define __defer_paste1(x, y) __defer_paste(x, y)
#define defer(fn) void __defer_paste1(__defer_fn_, __LINE__)(void) { preserve_errno({ fn; }); }; __attribute__((unused)) __attribute__((__cleanup__(__defer_cb))) void (*__defer_paste1(__defer_, __LINE__))(void) = __defer_paste1(__defer_fn_, __LINE__)

extern char **environ;

/** Like getenv, but gets the entire variable. */
static char *getenve(const char *name) {
    int i;
    size_t l = strlen(name);
    if (!environ || !*name || strchr(name, '=')) {
        return NULL;
    }
    for (i = 0; environ[i] && (strncmp(name, environ[i], l) || environ[i][l] != '='); i++) {
        continue;
    }
    if (environ[i]) {
        return environ[i];
    }
    return NULL;
}

/** Get the number of available logical cores. */
static int nprocs(void) {
    int c = get_nprocs_conf();
    cpu_set_t cs;
    CPU_ZERO(&cs);
    sched_getaffinity(0, sizeof(cs), &cs);
    return CPU_COUNT(&cs) < c ? CPU_COUNT(&cs) : c;
}

/** Replace the process cmdline. It must be initialized by calling it with a NULL fmt first. */
static __attribute__ ((__format__ (__printf__, 2, 3))) int setproctitle(char **argv, const char *fmt, ...) {
    // https://github.com/torvalds/linux/commit/d26d0cd97c88eb1a5704b42e41ab443406807810
    // https://source.chromium.org/chromium/chromium/src/+/master:content/common/set_process_title_linux.cc

    static int argv_len = 0;
    if (!fmt) {
        for (char **x = argv; *x; x++) {
            if (*x != *argv + argv_len) {
                // next arg is not consecutive, so stop here
                break;
            }
            argv_len += strlen(*x) + 1;
        }
        return argv_len;
    }

    va_list a;
    va_start(a, fmt);
    int r = vsnprintf(*argv, argv_len, fmt, a);
    for (char *x = *argv; x < *argv + argv_len; x++) {
        if (x == *argv + argv_len-1) {
            *x = '.';
        } else if (x >= *argv + r || x == *argv + argv_len-2) {
            *x = '\0';
        }
    }
    va_end(a);
    return r;
}

/** The current status of a Northstar server */
struct ns_status {
    char map_name[32];
    char playlist_name[32];
    int player_count;
    int max_players;
};

/**
 * Starts an Xvfb instance as a child and returns the display with pid_out set. Otherwise, -1 is returned with errno and
 * err_out set.
 */
static int xvfb(struct timespec timeout, int output_fd, pid_t *pid_out, char *err_out, size_t err_out_sz) {
    #define xvfb_err(c, x, fmt, ...) preserve_errno({ \
        errno = c;                                    \
        preserve_errno(x);                            \
        if (err_out && err_out_sz) preserve_errno({   \
            snprintf(err_out, err_out_sz, fmt, ##__VA_ARGS__); \
        });                                           \
    })

    int timerfd = timerfd_create(CLOCK_MONOTONIC, TFD_CLOEXEC|TFD_NONBLOCK);
    if (timerfd == -1) {
        xvfb_err(errno, {}, "timerfd_create: %m");
        return -1;
    }

    if (timerfd_settime(timerfd, 0, &(struct itimerspec){
        .it_value = timeout,
    }, NULL)) {
        xvfb_err(errno, {
            close(timerfd);
        }, "timerfd_settime: %m");
        return -1;
    }

    int displayfd[2];
    if (pipe2(displayfd, O_DIRECT|O_NONBLOCK)) {
        xvfb_err(errno, {
            close(timerfd);
        }, "pipe2: %m");
        return -1;
    }

    pid_t pid = fork();
    if (!pid) {
        setsid();
        dup2(open("/dev/null", O_RDONLY), 0);
        dup2(output_fd, 1);
        dup2(output_fd, 2);
        char x[16];
        snprintf(x, sizeof(x), "%d", displayfd[1]);
        close(displayfd[0]);
        execvp("Xvfb", (char*[]){"Xvfb", "-displayfd", x, "-nolisten", "tcp", "-nolisten", "unix", "-noreset", NULL});
        snprintf(x, sizeof(x), "E%d", errno);
        write(displayfd[1], x, strlen(x));
        _exit(127);
    }
    close(displayfd[1]);

    if (pid == -1) {
        return -1;
    }

    struct pollfd pfd[] = {
        { .fd = displayfd[0], .events = POLLIN },
        { .fd = timerfd, .events = POLLIN },
    };

    for (;;) {
        switch (poll(pfd, sizeof(pfd)/sizeof(*pfd), 100)) {
        case -1:
            if (errno == EINTR) {
                continue;
            }
            preserve_errno({
                close(displayfd[0]);
                close(timerfd);
                kill(pid, SIGKILL);
            });
            return -1;
        case 0:
            if (kill(pid, 0) == -1 && errno == ESRCH) {
                int st;
                if (waitpid(pid, &st, WNOHANG) == -1) {
                    xvfb_err(errno, {
                        close(displayfd[0]);
                        close(timerfd);
                    }, "xvfb exited, but failed to get status (%m)");
                    return -1;
                }
                if (WIFSIGNALED(st)) {
                    xvfb_err(EIO, {
                        close(displayfd[0]);
                        close(timerfd);
                    }, "xvfb killed with signal %d", WTERMSIG(st));
                } else {
                    xvfb_err(EIO, {
                        close(displayfd[0]);
                        close(timerfd);
                    }, "xvfb exited with status %d", WEXITSTATUS(st));
                }
                return -1;
            }
            continue;
        }
        for (size_t i = 0; i < sizeof(pfd)/sizeof(*pfd); i++) {
            if (pfd[i].revents & POLLIN) {
                if (pfd[i].fd == displayfd[0]) {
                    ssize_t n;
                    char buf[16];
                    if ((n = read(displayfd[0], buf, sizeof(buf)-1)) == -1) {
                        xvfb_err(errno, {
                            close(displayfd[0]);
                            close(timerfd);
                            kill(pid, SIGKILL);
                        }, "read displayfd: %m");
                        return -1;
                    }
                    buf[n] = '\0';

                    char *p = buf;
                    if (*p == 'E') {
                        p++;
                    }

                    char *e;
                    long x = strtol(p, &e, 10);
                    if (e == p) {
                        xvfb_err(EPROTO, {
                            close(displayfd[0]);
                            close(timerfd);
                            kill(pid, SIGKILL);
                        }, "parse displayfd (%.*s): %m", (int)(n), buf);
                        return -1;
                    }

                    if (*buf == 'E') {
                        xvfb_err(x, {
                            close(displayfd[0]);
                            close(timerfd);
                            kill(pid, SIGKILL);
                        }, "execvp Xvfb: %m");
                        return -1;
                    }

                    if (pid_out) {
                        *pid_out = pid;
                    }

                    // dirty workaround for #29 (Xvfb write a separate newline after the display number and exits if it can't)
                    nanosleep(&(struct timespec){
                        .tv_nsec = 50 * 1000 * 1000,
                    }, NULL);

                    close(displayfd[0]);
                    close(timerfd);
                    return x;
                }
                if (pfd[i].fd == timerfd) {
                    xvfb_err(errno, {
                        close(displayfd[0]);
                        close(timerfd);
                        kill(pid, SIGKILL);
                    }, "xvfb didn't initialize within %lds", (long)(timeout.tv_sec));
                    return -1;
                }
                abort();
            }
        }
    }
    #undef xvfb_err
}

static regex_t ns_status_re;

#define __NS_STATUS_RE_REGEXP(_r, _g) _r
#define NS_STATUS_RE_REGEXP NS_STATUS_RE(__NS_STATUS_RE_REGEXP, _, _)

#define __NS_STATUS_RE_GROUPS(_r, _g) do { _g } while (0)
#define NS_STATUS_RE_GROUPS(_int, _str) NS_STATUS_RE(__NS_STATUS_RE_GROUPS, _int, _str)

#define __NS_STATUS_RE_MATCHES_1(_r, _g) (1 _g)
#define __NS_STATUS_RE_MATCHES_2(_v) + 1
#define NS_STATUS_RE_MATCHES NS_STATUS_RE(__NS_STATUS_RE_MATCHES_1, __NS_STATUS_RE_MATCHES_2, __NS_STATUS_RE_MATCHES_2)

__attribute__((constructor)) static void ns_status_init(void) {
    #define x(_n, _r, _g) _r
    if (regcomp(&ns_status_re, NS_STATUS_RE_REGEXP, REG_EXTENDED)) {
        ns_log("failed to compile status regexp");
        abort();
    }
    #undef x
}

static int ns_status_parse(struct ns_status *st, const char *title) {
    regmatch_t m[NS_STATUS_RE_MATCHES];
    int rc;
    if ((rc = regexec(&ns_status_re, title, NS_STATUS_RE_MATCHES, m, 0))) {
        if (rc != REG_NOMATCH) {
            char err[512];
            regerror(rc, &ns_status_re, err, sizeof(err));
            ns_log("title regex match error: %s", err);
        }
        errno = EINVAL;
        return -1;
    }
    int i = 0;
    #define m_str(_v) \
        i++; snprintf(st->_v, sizeof(st->_v), "%.*s", (int)(m[i].rm_eo - m[i].rm_so), title + m[i].rm_so);
    #define m_int(_v) \
        i++; st->_v = 0; for (regoff_t j = m[i].rm_so; j < m[i].rm_eo; j++) { st->_v = 10 * st->_v + (title[j] - '0'); };
    NS_STATUS_RE_GROUPS(m_int, m_str);
    #undef m_str
    #undef m_int
    return 0;
}

static void ns_status_str(struct ns_status *st, char *buf, size_t buf_sz) {
    #define putf(fmt, ...) do { \
        if (buf_sz > 0) {       \
            int r = snprintf(buf, buf_sz, fmt, ##__VA_ARGS__); \
            if (r >= 0) {       \
                buf += r;       \
                buf_sz -= r;    \
            } else {            \
                return;         \
            }                   \
        }                       \
    } while (0)
    #define putft(cond, def, fmt, ...) do { \
        if (cond) {                         \
            putf(fmt, ##__VA_ARGS__);       \
        } else {                            \
            putf(def);                      \
        }                                   \
    } while (0)
    if (buf) {
        *buf = '\0';
        putft(st->player_count >= 0, "?", "%d", st->player_count);
        putft(st->max_players > 0, "/?", "/%d", st->max_players);
        putft(*st->map_name, " ???", " %s", st->map_name);
        putft(*st->playlist_name, " ???", " %s", st->playlist_name);
    }
    #undef putf
}

/** Captures console output, filters junk ANSI escapes from Wine, and catches title updates. */
struct ns_ioproc {
    struct {
        int fd_pty_master;
        int fd_pty_slave; // not CLOEXEC
        int state;
        size_t n_inp, n_tit, n_out;
        char b_inp[NS_IOPROC_OUTPUT_CHUNK_SIZE];
        char b_tit[NS_IOPROC_OUTPUT_CHUNK_SIZE + 1]; // +1 for the null terminator
        char b_out[NS_IOPROC_OUTPUT_CHUNK_SIZE * 2 + 32]; // b_inp + b_tit + room for unprocessed escapes
    } output;
    struct {
        int fd_pipe_title_r;
        int fd_pipe_title_w;
        char buf[NS_IOPROC_OUTPUT_CHUNK_SIZE+1];
    } title;
};

static int ns_ioproc_init(struct ns_ioproc *p) {
    int fd_pty_master = open("/dev/ptmx", O_RDWR | O_NOCTTY | O_CLOEXEC);
    if (fd_pty_master == -1) {
        preserve_errno({
            ns_perror_dbg("open pty master");
        });
        return -1;
    }

    if (ioctl(fd_pty_master, TIOCSPTLCK, &(int) {0}) == -1) {
        preserve_errno({
            ns_perror_dbg("unlock pty master");
            close(fd_pty_master);
        });
        return -1;
    }

    int fd_pty_slave_n;
    if (ioctl(fd_pty_master, TIOCGPTN, &fd_pty_slave_n) == -1) {
        preserve_errno({
            ns_perror_dbg("get pty number");
            close(fd_pty_master);
        });
        return -1;
    }

    char fd_pty_slave_path[20];
    snprintf(fd_pty_slave_path, sizeof(fd_pty_slave_path), "/dev/pts/%d", fd_pty_slave_n);

    int fd_pty_slave = open(fd_pty_slave_path, O_RDWR | O_NOCTTY); // not O_CLOEXEC
    if (fd_pty_slave == -1) {
        preserve_errno({
            ns_perror_dbg("open pty slave '%s'", fd_pty_slave_path);
            close(fd_pty_master);
        });
        return -1;
    }

    struct termios pty_termios;
    if (tcgetattr(fd_pty_slave, &pty_termios)) {
        preserve_errno({
            ns_perror_dbg("get pty slave termios");
            close(fd_pty_slave);
            close(fd_pty_master);
        });
        return -1;
    }

    pty_termios.c_iflag = BRKINT | IGNPAR | ISTRIP | IGNCR | IUTF8;
    pty_termios.c_oflag = OPOST | ONOCR;
    pty_termios.c_cflag = CREAD;
    pty_termios.c_lflag = ISIG | ICANON;

    // return from read() at least every 0.1s, whether or not data is available
    pty_termios.c_cc[VMIN] = 0;
    pty_termios.c_cc[VTIME] = 1;

    if (tcsetattr(fd_pty_slave, TCSANOW, &pty_termios)) {
        preserve_errno({
            ns_perror_dbg("set pty slave termios");
            close(fd_pty_slave);
            close(fd_pty_master);
        });
        return -1;
    }

    if (ioctl(fd_pty_slave, TIOCSWINSZ, &(struct winsize) {
        .ws_col = 1200,
        .ws_row = 25,
    })) {
        preserve_errno({
            ns_perror_dbg("set pty slave winsize");
            close(fd_pty_slave);
            close(fd_pty_master);
        });
        return -1;
    }

    int fd_pipe_title[2];
    if (pipe2(fd_pipe_title, O_DIRECT | O_NONBLOCK)) {
        preserve_errno({
            ns_perror_dbg("create title update pipe");
            close(fd_pty_slave);
            close(fd_pty_master);
        });
        return -1;
    }

    *p = (struct ns_ioproc){
        .output.fd_pty_master = fd_pty_master,
        .output.fd_pty_slave = fd_pty_slave,
        .title.fd_pipe_title_r = fd_pipe_title[0],
        .title.fd_pipe_title_w = fd_pipe_title[1],
    };
    return 0;
}

static void ns_ioproc_close(struct ns_ioproc *p) {
    close(p->output.fd_pty_slave);
    close(p->output.fd_pty_master);
    close(p->title.fd_pipe_title_w);
    close(p->title.fd_pipe_title_r);
}

static int ns_ioproc_output_pty(struct ns_ioproc *p) {
    return p->output.fd_pty_slave;
}

static int ns_ioproc_output_epoll_add(struct ns_ioproc *p, int fd) {
    return epoll_ctl(fd, EPOLL_CTL_ADD, p->output.fd_pty_master, &(struct epoll_event) {
        .events = EPOLLIN,
        .data.fd = p->output.fd_pty_master,
    });
}

static int ns_ioproc_output_epoll_check(struct ns_ioproc *p, struct epoll_event ev) {
    return ev.data.fd == p->output.fd_pty_master;
}

static const char *ns_ioproc_output_epoll_process(struct ns_ioproc *p, size_t *sz_out) {
    if (!sz_out) {
        errno = EINVAL;
        return NULL;
    }

    ssize_t tmp;
    while ((tmp = read(p->output.fd_pty_master, p->output.b_inp, sizeof(p->output.b_inp))) == -1) {
        if (errno == EWOULDBLOCK || errno == EAGAIN) {
            *sz_out = 0;
            return p->output.b_out;
        }
        if (errno != EINTR) {
            preserve_errno({
                ns_perror_dbg("read output pty");
            });
            return NULL;
        }
    }
    p->output.n_inp = (size_t)(tmp); // note: not EPOLLET, so we don't need to read it all at once; it'll just be triggered again later

    // fast path when no escape sequences in the buffer
    if (p->output.state == 0) {
        for (size_t i = 0; i < p->output.n_inp; i++) {
            if (p->output.b_inp[i] == 0x1B) {
                goto slow;
            }
        }
        *sz_out = p->output.n_inp;
        return p->output.b_inp;
    }

slow:
    p->output.n_out = 0;
    for (size_t i = 0; i < p->output.n_inp; i++) {
        char c = p->output.b_inp[i];
        switch (p->output.state) {
        case 0: // normal output
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = c;
                break;
            case 0x1B:
                p->output.state = 1;
                break;
            }
            break;
        case 1: // at \x1B
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = c;
                break;
            case ']':
                p->output.state = 2;
                break;
            case '[':
                p->output.state = 12;
                break;
            }
            break;
        case 2: // at \x1B]
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = ']';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case '0':
                p->output.state = 3;
                break;
            }
            break;
        case 3: // at \x1B]0
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = ']';
                p->output.b_out[p->output.n_out++] = '0';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case ';':
                p->output.state = 4;
                p->output.n_tit = 0;
                break;
            }
            break;
        case 4: // in \x1B]0;
            switch (c) {
            default:
                // next title char
                if (p->output.n_tit < NS_IOPROC_OUTPUT_CHUNK_SIZE) {
                    p->output.b_tit[p->output.n_tit++] = c;
                    break;
                }
                    __attribute__((fallthrough));
            case 0x07:
                // end of title || overflow
                if (p->output.n_tit == NS_IOPROC_OUTPUT_CHUNK_SIZE) {
                    p->output.state = 5;
                } else {
                    p->output.state = 0;
                }
                p->output.b_tit[p->output.n_tit] = '\0';
                if (write(p->title.fd_pipe_title_w, p->output.b_tit, p->output.n_tit) == -1) { // note: O_NONBLOCK
                    if (errno != EWOULDBLOCK) {
                        // ignore (and it shouldn't happen unless PIPE_BUF, which is usually 64kb, is full)
                    }
                    ns_perror_dbg("write title to pipe");
                    // ignore (it shouldn't happen, and there isn't much we can do about it if it does
                }
                p->output.n_tit = 0;
                break;
            case 0x1B:
                // start of a new escape sequence (this shouldn't happen)
                p->output.state = 1;
                break;
            }
            break;
        case 5: // in title
            switch (c) {
            default:
                // overflowing title character
                break;
            case 0x07:
                // end of the overflowing title
                p->output.state = 0;
                break;
            case 0x1B:
                // start of a new escape sequence (this shouldn't happen)
                p->output.state = 1;
                break;
            }
            break;
        case 12: // at \x1B[
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = '[';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case '?':
                p->output.state = 13;
                break;
            case '1':
                p->output.state = 23;
                break;
            case 'K':
                // ignore the CR equivalent
                p->output.state = 0;
                break;
            }
            break;
        case 13: // at \x1B[?
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = '[';
                p->output.b_out[p->output.n_out++] = '?';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case '2':
                p->output.state = 14;
                break;
            }
            break;
        case 14: // at \x1B[?2
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = '[';
                p->output.b_out[p->output.n_out++] = '?';
                p->output.b_out[p->output.n_out++] = '2';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case '5':
                p->output.state = 15;
                break;
            }
            break;
        case 15: // at \x1B[?25
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = '[';
                p->output.b_out[p->output.n_out++] = '?';
                p->output.b_out[p->output.n_out++] = '2';
                p->output.b_out[p->output.n_out++] = '5';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case 'l':
                // ignore hide cursor
                p->output.state = 0;
                break;
            case 'h':
                // ignore show cursor
                p->output.state = 0;
                break;
            }
            break;
        case 23: // at \x1B[1
            switch (c) {
            default:
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = 0x1B;
                p->output.b_out[p->output.n_out++] = '[';
                p->output.b_out[p->output.n_out++] = '1';
                p->output.b_out[p->output.n_out++] = c;
                break;
            case 'C':
                // move cursor right 1
                p->output.state = 0;
                p->output.b_out[p->output.n_out++] = ' ';
                break;
            }
            break;
        }
    }
    *sz_out = p->output.n_out;
    return p->output.b_out;
}

static int ns_ioproc_title_epoll_add(struct ns_ioproc *p, int fd) {
    return epoll_ctl(fd, EPOLL_CTL_ADD, p->title.fd_pipe_title_r, &(struct epoll_event) {
        .events = EPOLLIN,
        .data.fd = p->title.fd_pipe_title_r,
    });
}

static int ns_ioproc_title_epoll_check(struct ns_ioproc *p, struct epoll_event ev) {
    return ev.data.fd == p->title.fd_pipe_title_r;
}

static const char *ns_ioproc_title_epoll_process(struct ns_ioproc *p) {
    if (read(p->title.fd_pipe_title_r, p->title.buf, sizeof(p->title.buf)) == -1) {
        return NULL;
    }
    return p->title.buf;
}

/** Watches for server hangs by taking advantage of the title updates in the server loop. */
struct ns_watchdog {
    int timerfd;
    atomic_int init_ctr;
    atomic_long last_sec;
    int init_target;
    int init_timeout_sec;
    int interval_sec;
    char err[150];
};

/**
 * Initializes a ns_watchdog. The watchdog must receive init_target events within init_timeout_sec to start, then it
 * must receive at least one title update every interval_sec. If the timerfd can't be created, -1 is returned and errno
 * is set. Otherwise, 0 is returned.
 */
static int ns_watchdog_init(struct ns_watchdog *wd, int init_target, int init_timeout_sec, int interval_sec) {
    int timerfd = timerfd_create(CLOCK_MONOTONIC, TFD_CLOEXEC | TFD_NONBLOCK);
    if (timerfd == -1) {
        preserve_errno({
            ns_perror_dbg("create timerfd");
        });
        return -1;
    }
    if (timerfd_settime(timerfd, 0, &(struct itimerspec) {
        .it_value.tv_sec = init_timeout_sec,
    }, NULL) == -1) {
        preserve_errno({
            ns_perror_dbg("set timerfd");
            close(wd->timerfd);
        });
        return -1;
    }
    *wd = (struct ns_watchdog) {
        .timerfd = timerfd,
        .init_target = init_target,
        .init_timeout_sec = init_timeout_sec,
        .interval_sec = interval_sec,
    };
    return 0;
}

/**
 * Stops a ns_watchdog and frees the timerfd.
 */
static void ns_watchdog_stop(struct ns_watchdog *wd) {
    close(wd->timerfd);
}

/** Checks if the watchdog has received the initial ticks. */
static bool ns_watchdog_initialized(struct ns_watchdog *wd) {
    return atomic_load(&wd->init_ctr) >= wd->init_target;
}

/** Sends an update to the watchdog. Returns 0 on success or -1 with errno set. */
static int ns_watchdog_update(struct ns_watchdog *wd) {
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC_COARSE, &ts) == -1) {
        return -1;
    }
    if (!ns_watchdog_initialized(wd)) {
        long last = atomic_exchange(&wd->last_sec, (long)(ts.tv_sec));
        if (ts.tv_sec - last > wd->interval_sec) {
            atomic_store(&wd->init_ctr, 0);
            return 0;
        } else if (atomic_fetch_add(&wd->init_ctr, 1) < wd->init_target) {
            return 0;
        }
    }
    atomic_store(&wd->last_sec, ts.tv_sec);
    return timerfd_settime(wd->timerfd, 0, &(struct itimerspec) {
        .it_value.tv_sec = wd->interval_sec,
    }, NULL);
}

/** Adds the watchdog to the epoll file descriptor. */
static int ns_watchdog_epoll_add(struct ns_watchdog *wd, int fd) {
    return epoll_ctl(fd, EPOLL_CTL_ADD, wd->timerfd, &(struct epoll_event) {
        .events = EPOLLIN,
        .data.fd = wd->timerfd,
    });
}

/** Checks if an epoll event matches the watchdog. */
static bool ns_watchdog_epoll_check(struct ns_watchdog *wd, struct epoll_event ev) {
    return ev.data.fd == wd->timerfd;
}

/** Processes an epoll event and returns the error message, or NULL with errno set. */
static const char *ns_watchdog_epoll_process(struct ns_watchdog *wd) {
    uint64_t v;
    if (read(wd->timerfd, &v, sizeof(v)) == -1) {
        return NULL;
    }
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC_COARSE, &ts) == -1) {
        return NULL;
    }
    if (ns_watchdog_initialized(wd)) {
        snprintf(wd->err, sizeof(wd->err), "watchdog did not receive a title update in time: last tick was %lds ago", (long)(ts.tv_sec) - atomic_load(&wd->last_sec));
        return wd->err;
    } else {
        snprintf(wd->err, sizeof(wd->err), "watchdog did not receive enough title updates for initialization: only received %d/%d initial ticks less than %ds apart within %ds", atomic_load(&wd->init_ctr), wd->init_target, wd->interval_sec, wd->init_timeout_sec);
        return wd->err;
    }
}

int main(int argc, char **argv) {
    if (argc <= 1) {
        fprintf(stderr, "usage: %s game_dir [args...]\n", argc ? argv[0] : "nswrap");
        return 2;
    }

    if (geteuid() == 0) {
        ns_log("error: this program must not be run as root");
        return 1;
    }

    // init setproctitle and attempt to ensure there is a placeholder arg consisting of spaces
    for (const char *x = argv[argc - 1]; *x; x++) {
        if (*x != ' ') {
            char **nargv = alloca(argc + 2*sizeof(char*));
            for (int i = 0; i < argc; i++) {
                nargv[i] = argv[i];
            }
            nargv[argc] = "                                                                                                                                ";
            nargv[argc+1] = NULL;
            if (execve("/proc/self/exe", nargv, environ) == -1) {
                ns_perror("warning: self-exec with additional space in argv for process title failed: execve");
                argc++;
            }
            break;
        }
    }
    argc--;
    setproctitle(argv, NULL);
    argv[argc] = NULL;

    if (chdir(argv[1])) {
        ns_perror("error: chdir '%s'", argv[1]);
        return 1;
    }

    if (access("NorthstarLauncher.exe", F_OK)) {
        ns_log("error: NorthstarLauncher.exe missing");
        return 1;
    }

    int np = nprocs();
    struct sysinfo sinfo;
    struct utsname uinfo;
    if (sysinfo(&sinfo) == -1) {
        perror("warning: failed to get system info: sysinfo");
    } else if (uname(&uinfo) == -1) {
        perror("warning: failed to get system info: uname");
    } else {
        #ifdef NSWRAP_HASH
        #define NSWRAP_HASH__(x) #x
        #define NSWRAP_HASH_(x) NSWRAP_HASH__(x)
        ns_log("%s", NSWRAP_HASH_(NSWRAP_HASH));
        ns_log("");
        #undef NSWRAP_HASH_
        #undef NSWRAP_HASH__
        #endif
        ns_log("config");
        ns_log("  PATH=%s", getenv("PATH") ?: "(null)");
        ns_log("  HOME=%s", getenv("HOME") ?: "(null)");
        ns_log("  USER=%s", getenv("USER") ?: "(null)");
        ns_log("  HOSTNAME=%s", getenv("HOSTNAME") ?: "(null)");
        ns_log("  DISPLAY=%s", getenv("DISPLAY") ?: "(null)");
        ns_log("  WINEPREFIX=%s", getenv("WINEPREFIX") ?: "(null)");
        ns_log("  WINEDEBUG=%s", getenv("WINEDEBUG") ?: "(null)");
        ns_log("  WINESERVER=%s", getenv("WINESERVER") ?: "(null)");
        ns_log("");
        ns_log("system info:");
        ns_log("  kernel: %s %s %s %s %s", uinfo.sysname, uinfo.nodename, uinfo.release, uinfo.version, uinfo.machine);
        ns_log("  processor: %d cores", np);
        ns_log("  memory: %ld total, %ld free, %ld shared, %ld buffer", sinfo.totalram*sinfo.mem_unit, sinfo.freeram*sinfo.mem_unit, sinfo.sharedram*sinfo.mem_unit, sinfo.bufferram*sinfo.mem_unit);
        ns_log("  swap: %ld total, %ld free", sinfo.totalswap*sinfo.mem_unit, sinfo.freeswap*sinfo.mem_unit);
        ns_log("");
    }

    const char *wineprefix = getenv("WINEPREFIX");
    if (!wineprefix || *wineprefix != '/' || access(wineprefix, F_OK|R_OK|W_OK|X_OK)) {
        if (!wineprefix) {
            ns_log("error: WINEPREFIX not set");
        } else if (*wineprefix != '/') {
            ns_log("error: invalid WINEPREFIX '%s': not an absolute path", wineprefix);
        } else {
            ns_perror("error: invalid WINEPREFIX '%s'", wineprefix);
        }
        ns_log("note: the wineprefix must set HKCU\\Software\\Wine\\WineDbg\\ShowCrashDialog to DWORD:0, and HKCU\\Software\\Wine\\DllOverrides\\{mscoree,mshtml} to REG_SZ:\"\"");
        ns_log("note: optionally, it should also set HKLM\\System\\CurrentControlSet\\Services\\WineBus\\{DisableHidraw,DisableInput} to REG_DWORD:1 and HKCU\\Software\\Wine\\Drivers\\Audio to REG_SZ:\"\"");
        ns_log("note: if pg9182's d3d11 and gfsdk stubs are used, set HKCU\\Software\\Wine\\DllOverrides\\d3d11 to REG_SZ:\"native\" and HKCU\\Software\\Wine\\DllOverrides\\{d3d9,d3d10,d3d12,wined3d,winevulkan} to REG_SZ:\"\"");
        ns_log("note: each instance of nswrap should have its own prefix (to save space, you can symlink files in system32), but it's not (currently) required");
        ns_log("note: you can use the nswrap-wineprefix script to set up a new wineprefix");
        return 1;
    }

    if (!getenv("WINEDEBUG")) {
        ns_log("using WINEDEBUG='%s'", WINEDEBUG_DEFAULT);
    } else {
        ns_log("warning: WINEDEBUG has been overridden to '%s' (replacing the recommended value '%s')", getenv("WINEDEBUG"), WINEDEBUG_DEFAULT);
    }

    if (!getenv("DISPLAY")) {
        ns_log("warning: no X server running");
        ns_log("note: Xvfb is sufficient as long as you're using pg9182's d3d11 and gfsdk stubs");
    }

    if (np < NS_REQUIRED_CORES) {
        ns_log("warning: currently, at least %d cores are required, but only %d were found", NS_REQUIRED_CORES, np);
    }

    prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0);

    int fd_epoll = epoll_create1(EPOLL_CLOEXEC);
    if (fd_epoll == -1) {
        ns_perror("error: failed to create epoll fd");
        return 1;
    }
    defer(close(fd_epoll));

    int fd_pipe_errno[2];
    if (pipe2(fd_pipe_errno, O_DIRECT | O_NONBLOCK)) {
        ns_perror("error: failed to create errno pipe");
        return 1;
    }

    if (epoll_ctl(fd_epoll, EPOLL_CTL_ADD, fd_pipe_errno[0], &(struct epoll_event) {
        .events  = EPOLLIN,
        .data.fd = fd_pipe_errno[0],
    })) {
        ns_perror("error: failed to add errno pipe to epoll");
        return 1;
    }

    sigset_t mask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGINT);
    sigaddset(&mask, SIGTERM);
    sigaddset(&mask, SIGCHLD);

    int fd_signalfd = signalfd(-1, &mask, SFD_CLOEXEC | SFD_NONBLOCK);
    if (fd_signalfd == -1) {
        ns_perror("error: failed to register signal handlers: create signalfd");
        return 1;
    }
    defer(close(fd_signalfd));

    if (epoll_ctl(fd_epoll, EPOLL_CTL_ADD, fd_signalfd, &(struct epoll_event) {
        .events  = EPOLLIN,
        .data.fd = fd_signalfd,
    })) {
        ns_perror("error: failed to add signalfd to epoll");
        return 1;
    }

    int fd_timerfd_exit = timerfd_create(CLOCK_MONOTONIC, TFD_CLOEXEC | TFD_NONBLOCK);
    if (fd_timerfd_exit == -1) {
        ns_perror("error: failed to create exit timerfd");
        return 1;
    }
    defer(close(fd_timerfd_exit));

    if (epoll_ctl(fd_epoll, EPOLL_CTL_ADD, fd_timerfd_exit, &(struct epoll_event) {
        .events  = EPOLLIN,
        .data.fd = fd_timerfd_exit,
    })) {
        ns_perror("error: failed to add exit timerfd to epoll");
        return 1;
    }

    struct ns_watchdog st_watchdog;
    if (ns_watchdog_init(&st_watchdog, 10, 4 * 60, 60)) {
        ns_perror("error: failed to create watchdog");
        return 1;
    }
    defer(ns_watchdog_stop(&st_watchdog));

    if (ns_watchdog_epoll_add(&st_watchdog, fd_epoll)) {
        ns_perror("error: failed to add watchdog to epoll");
        return 1;
    }

    struct ns_ioproc st_ioproc;
    if (ns_ioproc_init(&st_ioproc)) {
        ns_perror("error: failed to init i/o processor");
        return 1;
    }
    defer(ns_ioproc_close(&st_ioproc));

    if (ns_ioproc_output_epoll_add(&st_ioproc, fd_epoll)) {
        ns_perror("error: failed to add output pty to epoll");
        return 1;
    }

    if (ns_ioproc_title_epoll_add(&st_ioproc, fd_epoll)) {
        ns_perror("error: failed to add title pipe to epoll");
        return 1;
    }

    if (prctl(PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)) {
        ns_log("warning: failed to set the child subreaper; processes will not be reaped");
    }

    pid_t xvfb_pid = -1;
    if (getenv("DISPLAY") && !strcmp(getenv("DISPLAY"), "xvfb")) {
        ns_log("starting xvfb");

        char buf[256];

        int display = xvfb((struct timespec) {
            .tv_sec = 3,
        }, 1, &xvfb_pid, buf, sizeof(buf));
        if (display == -1) {
            ns_log("error: failed to start xvfb: %s", buf);
            return 1;
        }

        snprintf(buf, sizeof(buf), ":%d", display);
        ns_log("xvfb started on display %s with pid %d", buf, xvfb_pid);
        setenv("DISPLAY", buf, 1);
    }
    defer({
        if (xvfb_pid != -1) {
            ns_log("killing xvfb");
            kill(xvfb_pid, SIGKILL);
        }
    });

    ns_log("starting wine");

    int wine_argv_n = 0;
    char **wine_argv = alloca(sizeof(char **) * (argc + 2)); // args (replacing 0 with wine64) + -dedicated + terminator

    wine_argv[wine_argv_n++] = "wine64";
    wine_argv[wine_argv_n++] = "NorthstarLauncher.exe";
    wine_argv[wine_argv_n++] = "-dedicated";

    for (int i = 2; i < argc; i++) {
        wine_argv[wine_argv_n++] = argv[i];
    }
    wine_argv[wine_argv_n] = NULL;

    char **wine_envp = (char *[]) {
        getenve("PATH") ?: "PATH=/usr/local/bin:/bin:/usr/bin",
        getenve("HOSTNAME") ?: "HOSTNAME=none",
        getenve("HOME") ?: "HOME=/",
        getenve("USER") ?: "USER=none",
        getenve("WINEDEBUG") ?: "WINEDEBUG=" WINEDEBUG_DEFAULT,
        getenve("WINEPREFIX"), // will not be null; already checked
        getenve("DISPLAY"), // will not be null; already checked
        getenve("WINESERVER"), // may be null
        NULL,
    };

    int fd_pty_slave = ns_ioproc_output_pty(&st_ioproc);

    pid_t wine_pid = fork();
    if (!wine_pid) {
        setsid();
        ioctl(fd_pty_slave, TIOCSCTTY, 0);
        dup2(fd_pty_slave, 0);
        dup2(fd_pty_slave, 1);
        dup2(fd_pty_slave, 2);
        close(fd_pty_slave);
        close(fd_pipe_errno[0]);
        execvpe(wine_argv[0], (char *const *) (wine_argv), (char *const *) (wine_envp));
        int n = errno;
        write(fd_pipe_errno[1], &n, sizeof(n));
        close(fd_pipe_errno[1]);
        _exit(127);
    }

    if (sigprocmask(SIG_BLOCK, &mask, NULL) == -1) {
        ns_perror("error: failed to register signal handlers: mask signals");
        kill(wine_pid, SIGKILL);
        return 1;
    }

    const char *nswrap_title = getenv("NSWRAP_TITLE");
    if (nswrap_title) {
        if (*nswrap_title) {
            setproctitle(argv, "northstar %s", nswrap_title);
        }
    } else {
        setproctitle(argv, "northstar");
    }

    bool st_exiting = false;
    uint64_t st_last_title_update = 0;
    bool st_shown_title_warning = false;

    for (;;) {
        struct epoll_event evt;
        if (epoll_wait(fd_epoll, &evt, 1, -1) == -1) {
            if (errno != EINTR) {
                ns_perror("error: epoll");
                goto cleanup;
            }
            continue;
        }
        if (evt.data.fd == fd_signalfd) {
            struct signalfd_siginfo siginfo;
            if (read(fd_signalfd, &siginfo, sizeof(siginfo)) == -1) {
                ns_perror("error: process events: read signalfd");
                goto cleanup;
            }
            switch (siginfo.ssi_signo) {
            case SIGINT:
            case SIGTERM:
                if (st_exiting) {
                    ns_log("killing process");
                    goto cleanup;
                } else {
                    st_exiting = true;
                }
                if (siginfo.ssi_signo == SIGINT) {
                    ns_log("received SIGINT; waiting for server to exit (press ctrl-c again to kill)");
                } else {
                    ns_log("received SIGTERM; waiting for server to exit");
                }
                if (timerfd_settime(fd_timerfd_exit, 0, &(struct itimerspec) {
                    .it_value.tv_sec = 4,
                }, NULL)) {
                    ns_perror("error: failed to set exit timer\n");
                    goto cleanup;
                }
                if (kill(wine_pid, SIGTERM)) {
                    ns_log("warning: failed to send SIGTERM to pid %ld", (long) (wine_pid));
                }
                break;
            case SIGCHLD:
                if (siginfo.ssi_code == CLD_EXITED || siginfo.ssi_code == CLD_KILLED || siginfo.ssi_code == CLD_DUMPED) {
                    if ((pid_t)(siginfo.ssi_pid) == wine_pid) {
                        // note: the process will be reaped later
                        goto cleanup;
                    } else if ((pid_t)(siginfo.ssi_pid) == xvfb_pid) {
                        if (siginfo.ssi_code == CLD_EXITED) {
                            ns_log("warning: xvfb terminated: exited with status %d", siginfo.ssi_status);
                        } else if (siginfo.ssi_code == CLD_KILLED) {
                            ns_log("warning: xvfb terminated: killed by signal %d", siginfo.ssi_status);
                        } else if (siginfo.ssi_code == CLD_DUMPED) {
                            ns_log("warning: xvfb dumped core");
                        }
                        xvfb_pid = -1;
                        waitpid(siginfo.ssi_pid, NULL, WNOHANG); // reap the process
                    } else {
                        waitpid(siginfo.ssi_pid, NULL, WNOHANG); // reap the process
                        //ns_log("debug: reaped child %ld", (long) (siginfo.ssi_pid));
                    }
                }
                break;
            default:
                ns_log("warning: unexpected signal %d; ignoring", siginfo.ssi_signo);
                break;
            }
            continue;
        }
        if (ns_watchdog_epoll_check(&st_watchdog, evt)) {
            const char *err = ns_watchdog_epoll_process(&st_watchdog);
            if (!err) {
                ns_perror("error: watchdog is buggy");
                goto cleanup;
            }
            if (ns_watchdog_initialized(&st_watchdog)) {
                ns_log("error: watchdog: %s", err);
                goto cleanup;
            } else {
                ns_log("warning: watchdog: %s", err);
                continue;
            }
        }
        if (ns_ioproc_output_epoll_check(&st_ioproc, evt)) {
            size_t output_sz;
            const char *output = ns_ioproc_output_epoll_process(&st_ioproc, &output_sz);
            if (!output) {
                ns_perror("error: failed to process i/o");
                goto cleanup;
            }
            if (output_sz) {
                fwrite(output, 1, output_sz, stdout);
                fflush(stdout);
            }
            continue;
        }
        if (ns_ioproc_title_epoll_check(&st_ioproc, evt)) {
            const char *title = ns_ioproc_title_epoll_process(&st_ioproc);
            if (!title) {
                ns_perror("error: failed to process title update");
                goto cleanup;
            }
            if (*title) {
                if (ns_watchdog_update(&st_watchdog) == -1) {
                    ns_perror("error: failed to update watchdog");
                    goto cleanup;
                }
                if (!(nswrap_title && !*nswrap_title)) {
                    struct timespec ts;
                    if (clock_gettime(CLOCK_MONOTONIC_COARSE, &ts)) {
                        ns_perror("error: failed to get current CLOCK_MONOTONIC_COARSE time");
                        goto cleanup;
                    }
                    uint64_t tt = ts.tv_sec * 10 + ts.tv_nsec / 100000000; // deciseconds
                    if (tt - st_last_title_update > 2) {
                        struct ns_status st;
                        if (ns_status_parse(&st, title)) {
                            if (!st_shown_title_warning) {
                                ns_log(
                                    "failed to parse title '%s'; status information will not be visible in the process list",
                                    title);
                                st_shown_title_warning = true;
                            }
                            if (nswrap_title) {
                                setproctitle(argv, "northstar %s", nswrap_title);
                            } else {
                                setproctitle(argv, "northstar");
                            }
                        } else {
                            char sts[512];
                            ns_status_str(&st, sts, sizeof(sts));
                            if (nswrap_title) {
                                setproctitle(argv, "northstar %s [%s]", nswrap_title, sts);
                            } else {
                                setproctitle(argv, "northstar [%s]", sts);
                            }
                            st_shown_title_warning = false;
                        }
                        st_last_title_update = tt;
                    }
                }
            }
            continue;
        }
        if (evt.data.fd == fd_timerfd_exit) {
            ns_log("warning: process did not exit in time; killing it");
            goto cleanup;
        }
        if (evt.data.fd == fd_pipe_errno[0]) {
            int n;
            if (read(fd_pipe_errno[0], &n, sizeof(n)) == -1) {
                ns_perror("error: exec '%s' failed, but we couldn't read the error", wine_argv[0]);
            } else {
                ns_log("error: exec '%s' failed: %s", wine_argv[0], strerror(n));
            }
            return 1;
        }
        ns_log("error: process events: unhandled fd %d", evt.data.fd);
        goto cleanup;
    }

cleanup:
    fflush(stdout);
    fflush(stderr);

    // get the wine exit status, but kill it first if it's still running
    siginfo_t siginfo = {};
    if (waitid(P_PID, wine_pid, &siginfo, WEXITED|WNOHANG) == -1 || siginfo.si_pid == 0) {
        ns_log("killing wine");
        if (kill(wine_pid, SIGKILL) == -1) {
            ns_perror("error: failed to kill wine");
        }
        for (int i = 0; siginfo.si_pid == 0; i++) {
            if (i > 10) {
                ns_perror("error: failed to get northstar exit status");
                break;
            }
            if (waitid(P_PID, wine_pid, &siginfo, WEXITED|WNOHANG) == -1) {
                break;
            }
            nanosleep(&(struct timespec){
                .tv_nsec = 100 * 1000 * 1000,
            }, NULL);
        }
    }
    if (siginfo.si_pid == 0) {
        // this should never happen
        ns_log("error: failed to get northstar exit status: did not exit even after killed");
    } else if (siginfo.si_code == CLD_KILLED) {
        ns_log("northstar killed by signal %d", siginfo.si_status);
    } else if (siginfo.si_code == CLD_EXITED) {
        if (siginfo.si_status == 127) {
            ns_log("northstar failed to start");
        } else {
            ns_log("northstar exited with status %d", siginfo.si_status);
        }
    } else if (siginfo.si_code == CLD_DUMPED) {
        ns_log("northstar dumped core");
    }

    // kill xvfb if it's still running
    if (xvfb_pid != -1 && kill(xvfb_pid, 0) == 0 && errno != ESRCH) {
        ns_log("killing xvfb");
        kill(xvfb_pid, SIGKILL);
        xvfb_pid = -1;
    }

    // the default wineserver timeout is 3s, so wait up to 5s for all children to exit
    ns_log("waiting for children to exit");
    struct timespec ts, tc;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    for (;;) {
        switch (waitpid(-1, NULL, WNOHANG)) {
        case -1:
            if (errno != EINTR) {
                if (errno != ECHILD) {
                    ns_perror("error: failed to reap remaining children to exit");
                }
                return 1;
            }
            continue; // try again immediately
        case 0:
            clock_gettime(CLOCK_MONOTONIC, &tc);
            if (tc.tv_sec - ts.tv_sec > 4) {
                ns_log("warning: children did not exit in time");
                return 1;
            }
            break; // no children to wait for
        default:
            continue; // child reaped; try another one immediately
        }
        nanosleep(&(struct timespec){
            .tv_nsec = 100 * 1000 * 1000,
        }, NULL);
    }
}
