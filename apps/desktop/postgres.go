package main

// Embedded Postgres lifecycle, owned end-to-end.
//
// Why not fergusstrange/embedded-postgres: it execs initdb/pg_ctl with no
// console suppression, so on Windows every launch popped stray console
// windows (the parent is GUI-subsystem, children are console-subsystem and
// Windows allocates each a fresh console). Its os.Stat("bin/pg_ctl") check
// also never matches pg_ctl.exe, so the ~80MB runtime re-extracted on every
// start. Owning these ~250 lines buys: hidden children on every path, a
// one-time extract, real progress for the setup UI, and OS-level child
// cleanup (Job Object / Pdeathsig) so a crashed cal.exe can't orphan postgres.
//
// Bundled runtime: when the installer ships <exe>/pg-runtime the first run
// needs no network at all. Otherwise the binaries download once into
// <dataDir>/pg-bin and are reused.

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/xi2/xz"
)

// pgVersion tracks the zonky embedded-postgres binary release.
// Keep in sync with PG_VERSION in scripts/prepare-pg-runtime.mjs.
const pgVersion = "18.3.0"
const pgBinaryRepo = "https://repo1.maven.org/maven2"

// progress reports (phase, human detail) to the setup screen.
type progress func(phase, detail string)

// embeddedPG owns one postgres child process and its log handle.
type embeddedPG struct {
	binDir  string // postgres binaries (bundled pg-runtime or downloaded pg-bin)
	dataDir string // the PG data cluster
	port    uint32
	cmd     *exec.Cmd
	done    chan struct{} // closed by the single waiter goroutine on exit
	job     jobHandle     // Windows Job Object; zero elsewhere
	logFile *os.File
}

func pgBin(binDir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, "bin", name)
}

func hasPostgresRuntime(binDir string) bool {
	_, err := os.Stat(pgBin(binDir, "postgres"))
	return err == nil
}

// bundledRuntimeDir returns the installer-shipped runtime next to the
// executable, or "" when running unpackaged (dev, portable binary).
func bundledRuntimeDir() string {
	if dir := os.Getenv("CAL_PG_RUNTIME"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exe)
	// NSIS installs pg-runtime/ as a subdirectory; tolerate a flat layout too
	// (bin/lib/share directly beside the exe) in case a packager flattens it.
	for _, dir := range []string{filepath.Join(exeDir, "pg-runtime"), exeDir} {
		if hasPostgresRuntime(dir) {
			return dir
		}
	}
	return ""
}

// startEmbeddedPostgres returns a running Postgres and its DSN.
// Sibling of the old library call; data stays at <dataDir>/pg either way.
func startEmbeddedPostgres(dataDir string, report progress, logFile *os.File) (*embeddedPG, string, error) {
	binDir := bundledRuntimeDir()
	if binDir == "" {
		binDir = filepath.Join(dataDir, "pg-bin")
	}

	pg := &embeddedPG{
		binDir:  binDir,
		dataDir: filepath.Join(dataDir, "pg"),
		logFile: logFile,
	}

	if !hasPostgresRuntime(binDir) {
		report("download", "Downloading database engine — first run only")
		if err := downloadPostgres(binDir, report); err != nil {
			return nil, "", fmt.Errorf("postgres download: %w", err)
		}
	}

	port, err := freePort()
	if err != nil {
		return nil, "", err
	}
	pg.port = port

	if err := pg.start(report); err != nil {
		return nil, "", err
	}

	dsn := fmt.Sprintf("postgres://cal:cal@127.0.0.1:%d/cal?sslmode=disable", port)
	return pg, dsn, nil
}

// start: stale-lock cleanup, initdb on first run, spawn, readiness poll.
func (p *embeddedPG) start(report progress) error {
	if err := p.stopStalePostmaster(); err != nil {
		return err
	}

	fresh := false
	if _, err := os.Stat(filepath.Join(p.dataDir, "PG_VERSION")); os.IsNotExist(err) {
		report("initdb", "Creating database — first run only")
		if err := p.initDB(); err != nil {
			return fmt.Errorf("initdb: %w", err)
		}
		fresh = true
	}

	report("postgres", "Starting database")
	p.cmd = exec.Command(pgBin(p.binDir, "postgres"),
		"-D", p.dataDir,
		"-p", strconv.FormatUint(uint64(p.port), 10),
		"-c", "listen_addresses=127.0.0.1",
	)
	p.cmd.Stdout = p.logFile
	p.cmd.Stderr = p.logFile
	hideConsole(p.cmd)

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("postgres start: %w", err)
	}
	p.job, _ = killWithParent(p.cmd) // best-effort; stale-pid recovery covers the rest

	p.done = make(chan struct{})
	go func() {
		_ = p.cmd.Wait()
		close(p.done)
	}()

	if err := p.waitReady(45 * time.Second); err != nil {
		p.stop()
		return fmt.Errorf("postgres did not become ready: %w\ntail of log:\n%s", err, p.logTail())
	}

	if fresh {
		// initdb only makes postgres/template0/template1 — the app DB is ours.
		report("database", "Creating application database")
		if err := p.createAppDB(); err != nil {
			p.stop()
			return fmt.Errorf("create database: %w", err)
		}
	}
	return nil
}

// createAppDB creates the "cal" database once, after a fresh initdb.
func (p *embeddedPG) createAppDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, fmt.Sprintf(
		"postgres://cal:cal@127.0.0.1:%d/postgres?sslmode=disable", p.port))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE DATABASE "cal"`)
	return err
}

// stop tries graceful fast shutdown, then hard-kills the child.
func (p *embeddedPG) stop() {
	defer closeJob(p.job)
	if p.cmd == nil || p.cmd.Process == nil || p.done == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = gracefulStop(ctx, p)
	select {
	case <-p.done:
	case <-time.After(10 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

// stopStalePostmaster: if a previous crash left a live postmaster holding the
// data dir, shut it down so this launch can take the lock. Dead pidfiles are
// left alone — postgres removes them itself.
func (p *embeddedPG) stopStalePostmaster() error {
	pidFile := filepath.Join(p.dataDir, "postmaster.pid")
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return nil
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || pid <= 0 {
		return nil
	}
	if !pidIsPostmaster(pid) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, pgBin(p.binDir, "pg_ctl"),
		"stop", "-D", p.dataDir, "-m", "immediate", "-w")
	cmd.Stdout = p.logFile
	cmd.Stderr = p.logFile
	hideConsole(cmd)
	_ = cmd.Run()
	return nil
}

func (p *embeddedPG) initDB() error {
	_ = os.RemoveAll(p.dataDir)
	if err := os.MkdirAll(p.dataDir, 0700); err != nil {
		return err
	}

	pwFile := filepath.Join(p.dataDir, "..", "pg-init-pw")
	if err := os.WriteFile(pwFile, []byte("cal"), 0600); err != nil {
		return err
	}
	defer os.Remove(pwFile)

	cmd := exec.Command(pgBin(p.binDir, "initdb"),
		"-A", "password",
		"-U", "cal",
		"-D", p.dataDir,
		"--pwfile", pwFile,
		"-E", "UTF-8",
		"--locale=C",
		"--no-sync",
	)
	cmd.Stdout = p.logFile
	cmd.Stderr = p.logFile
	hideConsole(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\n%s", err, p.logTail())
	}
	return nil
}

// waitReady polls with real connections — the listen socket opens before
// postgres finishes startup (57P03 "starting up"), so a bare TCP dial can
// report ready too early and break createAppDB on a race.
func (p *embeddedPG) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	dsn := fmt.Sprintf("postgres://cal:cal@127.0.0.1:%d/postgres?sslmode=disable&connect_timeout=2", p.port)
	for time.Now().Before(deadline) {
		select {
		case <-p.done:
			return fmt.Errorf("postgres exited with %s", p.cmd.ProcessState)
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := pgx.Connect(ctx, dsn)
		if err == nil {
			_ = conn.Close(ctx)
			cancel()
			return nil
		}
		cancel()
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s", timeout)
}

func freePort() (uint32, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return uint32(ln.Addr().(*net.TCPAddr).Port), nil
}

// downloadPostgres fetches the zonky binary jar, verifies its checksum and
// extracts the embedded .txz into binDir.
func downloadPostgres(binDir string, report progress) error {
	osName, arch := pgArchiveTarget()
	url := fmt.Sprintf("%s/io/zonky/test/postgres/embedded-postgres-binaries-%s-%s/%s/embedded-postgres-binaries-%s-%s-%s.jar",
		pgBinaryRepo, osName, arch, pgVersion, osName, arch, pgVersion)

	jar, err := fetchURL(url, report)
	if err != nil {
		return err
	}

	if err := verifySHA256(url, jar); err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(jar), int64(len(jar)))
	if err != nil {
		return fmt.Errorf("postgres jar: %w", err)
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".txz") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			return extractTxz(rc, binDir, report)
		}
	}
	return fmt.Errorf("no .txz inside %s", url)
}

// pgArchiveTarget mirrors embedded-postgres' version_strategy.go for the
// platforms this app ships: windows/amd64+arm64, linux/amd64+arm64,
// darwin/arm64.
func pgArchiveTarget() (osName, arch string) {
	osName, arch = runtime.GOOS, runtime.GOARCH
	if osName == "linux" {
		switch arch {
		case "arm64":
			arch = "arm64v8"
		case "arm":
			arch = "arm32v6"
		}
		if _, err := os.Stat("/etc/alpine-release"); err == nil {
			arch += "-alpine"
		}
	}
	if osName == "darwin" && arch == "arm64" {
		arch = "arm64v8" // pgVersion >= 14.2, so the arm build exists
	}
	return osName, arch
}

// pgHTTP bounds the whole fetch (10 min floor for ~80MB on slow links) and
// stalls on headers — a bare http.Get can hang init forever.
var pgHTTP = &http.Client{
	Timeout:   10 * time.Minute,
	Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second},
}

func fetchURL(url string, report progress) ([]byte, error) {
	resp, err := pgHTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}

	var buf bytes.Buffer
	var n, total int64
	total = resp.ContentLength
	w := io.MultiWriter(&buf, writerFunc(func(p []byte) (int, error) {
		n += int64(len(p))
		if total > 0 {
			report("download", fmt.Sprintf("Downloading database engine — %d%%", 100*n/total))
		}
		return len(p), nil
	}))
	if _, err := io.Copy(w, resp.Body); err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	return buf.Bytes(), nil
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func verifySHA256(url string, body []byte) error {
	resp, err := pgHTTP.Get(url + ".sha256")
	if err != nil {
		return nil // checksum endpoint unreachable — match upstream behaviour
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	sum, _ := io.ReadAll(resp.Body)
	got := sha256.Sum256(body)
	if hex.EncodeToString(got[:]) != strings.TrimSpace(string(sum)) {
		return fmt.Errorf("postgres archive checksum mismatch")
	}
	return nil
}

// extractTxz streams the tar.xz payload into dst, preserving file modes.
func extractTxz(r io.Reader, dst string, report progress) error {
	xr, err := xz.NewReader(r, 0)
	if err != nil {
		return fmt.Errorf("xz reader: %w", err)
	}
	tr := tar.NewReader(xr)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dst), "pg-extract-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	n := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		name := strings.TrimPrefix(hdr.Name, "./")
		if name == "" || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue // path traversal guard
		}
		target := filepath.Join(tmp, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)&0777); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			_ = f.Close()
			if n++; n%200 == 0 {
				report("extract", fmt.Sprintf("Unpacking database engine — %d files", n))
			}
		case tar.TypeSymlink:
			if runtime.GOOS != "windows" {
				_ = os.Remove(target)
				if err := os.Symlink(hdr.Linkname, target); err != nil {
					return err
				}
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	_ = os.RemoveAll(dst)
	return os.Rename(tmp, dst)
}

// logTail returns the end of the pg log for error surfacing; "" when the log
// file is nil (tests) or unreadable.
func (p *embeddedPG) logTail() string {
	if p.logFile == nil {
		return ""
	}
	_ = p.logFile.Sync()
	return tailFile(p.logFile.Name(), 4<<10)
}

func tailFile(path string, max int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	if st.Size() > max {
		_, _ = f.Seek(-max, io.SeekEnd)
	}
	b, _ := io.ReadAll(f)
	return strings.TrimSpace(string(b))
}
