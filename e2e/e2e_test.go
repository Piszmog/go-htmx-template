//go:build e2e

package e2e_test

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	_ "modernc.org/sqlite"
)

// global variables, can be used in any tests.
var (
	pw           *playwright.Playwright
	browser      playwright.Browser
	expect       playwright.PlaywrightAssertions
	isChromium   bool
	isFirefox    bool
	isWebKit     bool
	browserName  = getBrowserName()
	browserType  playwright.BrowserType
	app          *exec.Cmd
	baseURL      *url.URL
	rateLimitApp *exec.Cmd
	rateLimitURL string
)

// defaultContextOptions for most tests.
var defaultContextOptions = playwright.BrowserNewContextOptions{
	AcceptDownloads: playwright.Bool(true),
	HasTouch:        playwright.Bool(true),
}

// serverLog buffers lines from the app server's stdout/stderr.
// Lines are only printed when the test suite fails, keeping passing runs quiet.
type serverLog struct {
	mu    sync.Mutex
	lines []string
}

func (s *serverLog) append(line string) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	s.mu.Unlock()
}

func (s *serverLog) dump() {
	s.mu.Lock()
	lines := make([]string, len(s.lines))
	copy(lines, s.lines)
	s.mu.Unlock()
	for _, line := range lines {
		fmt.Println(line)
	}
}

var srvLog serverLog

// scannerWg tracks all running pipe-scanner goroutines so TestMain can wait for them
// to finish draining before calling srvLog.dump().
var scannerWg sync.WaitGroup

// pipeToLog reads lines from r and buffers them into srvLog prefixed with [tag].
func pipeToLog(r io.Reader, tag string) {
	scannerWg.Add(1)
	go func() {
		defer scannerWg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			srvLog.append("[" + tag + "] " + scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			srvLog.append("[" + tag + "-ERR] " + err.Error())
		}
	}()
}

// startWithPipes wires stdout and stderr pipes to srvLog then starts cmd.
func startWithPipes(cmd *exec.Cmd, stdoutTag, stderrTag string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	pipeToLog(stdout, stdoutTag)
	pipeToLog(stderr, stderrTag)
	return nil
}

// killAndWait sends SIGKILL to the process group of cmd and reaps the process.
func killAndWait(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		fmt.Println(err)
	}
	_ = cmd.Wait()
}

func TestMain(m *testing.M) {
	beforeAll()
	code := m.Run()
	afterAll()
	scannerWg.Wait()
	if code != 0 {
		fmt.Println("=== Server logs (dumped due to test failure) ===")
		srvLog.dump()
	}
	os.Exit(code)
}

// beforeAll prepares the environment, including
//   - start Playwright driver
//   - launch browser depends on BROWSER env
//   - init web-first assertions, alias as `expect`
func beforeAll() {
	err := playwright.Install()
	if err != nil {
		log.Fatalf("could not install Playwright: %v", err)
	}

	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	if browserName == "chromium" || browserName == "" {
		browserType = pw.Chromium
	} else if browserName == "firefox" {
		browserType = pw.Firefox
	} else if browserName == "webkit" {
		browserType = pw.WebKit
	}
	// launch browser, headless or not depending on HEADFUL env
	browser, err = browserType.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(os.Getenv("HEADFUL") == ""),
	})
	if err != nil {
		log.Fatalf("could not launch: %v", err)
	}
	// init web-first assertions with 5s timeout; 3s was too tight under parallel test load
	expect = playwright.NewPlaywrightAssertions(5000)
	isChromium = browserName == "chromium" || browserName == ""
	isFirefox = browserName == "firefox"
	isWebKit = browserName == "webkit"

	if err = buildCSS(); err != nil {
		log.Fatalf("could not build CSS: %v", err)
	}

	// build app binary once, reused by startApp and startRateLimitApp
	if err = buildApp(); err != nil {
		log.Fatalf("could not build app: %v", err)
	}

	if err = migrateDB("../test-db.sqlite3"); err != nil {
		log.Fatalf("could not migrate db: %v", err)
	}

	if err = startApp(); err != nil {
		log.Fatalf("could not start app: %v", err)
	}

	if err = waitForHealthCheck(baseURL.String()); err != nil {
		log.Fatalf("app failed health check: %v", err)
	}

	if err = seedDB(); err != nil {
		log.Fatalf("could not seed db: %v", err)
	}

	if err = migrateDB("../test-rl-db.sqlite3"); err != nil {
		log.Fatalf("could not migrate rate-limit db: %v", err)
	}

	if err = startRateLimitApp(); err != nil {
		log.Fatalf("could not start rate limit test app: %v", err)
	}
}

func buildApp() error {
	if err := os.MkdirAll("../tmp", 0750); err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	cmd := exec.Command("go", "build", "-o", "./tmp/e2e-server", "./cmd/server")
	cmd.Dir = "../"
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build: %w\n%s", err, out)
	}
	return nil
}

func buildCSS() error {
	cmd := exec.Command("go", "tool", "go-tw",
		"-i", "./styles/input.css",
		"-o", "./internal/dist/assets/css/output@dev.css",
	)
	cmd.Dir = "../"
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go-tw: %w\n%s", err, out)
	}
	return nil
}

func startApp() error {
	port := getPort()

	app = exec.Command("./tmp/e2e-server")
	app.Dir = "../"
	app.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	app.Env = append(
		os.Environ(),
		"DB_URL=./test-db.sqlite3",
		fmt.Sprintf("PORT=%d", port),
		"LOG_LEVEL=DEBUG",
		"RATE_LIMIT=1000000",
	)

	var err error
	baseURL, err = url.Parse(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return err
	}

	if err := startWithPipes(app, "STDOUT", "STDERR"); err != nil {
		return err
	}
	fmt.Printf("Started app on port %d, pid %d\n", port, app.Process.Pid)
	return nil
}

// startRateLimitApp starts a dedicated server with a low rate limit for use by
// rate-limiting tests. This keeps the main test server's bucket clean so
// unrelated tests are never unexpectedly throttled.
func startRateLimitApp() error {
	port := getPort()
	rateLimitApp = exec.Command("./tmp/e2e-server")
	rateLimitApp.Dir = "../"
	rateLimitApp.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	rateLimitApp.Env = append(
		os.Environ(),
		"DB_URL=./test-rl-db.sqlite3",
		fmt.Sprintf("PORT=%d", port),
		"LOG_LEVEL=ERROR",
		"RATE_LIMIT=30",
	)

	if err := startWithPipes(rateLimitApp, "RL-STDOUT", "RL-STDERR"); err != nil {
		return err
	}

	rateLimitURL = fmt.Sprintf("http://localhost:%d", port)
	return waitForHealthCheck(rateLimitURL)
}

// waitForHealthCheck polls the /health endpoint until it responds successfully.
// Returns nil on success, error on timeout or other failure.
func waitForHealthCheck(baseURL string) error {
	healthURL := baseURL + "/health"
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	for {
		select {
		case <-timeout:
			return fmt.Errorf("health check timeout: app did not become ready within 10s")
		case <-ticker.C:
			resp, err := client.Get(healthURL)
			if err != nil {
				continue
			}

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}

				bodyStr := string(body)
				if strings.Contains(bodyStr, `"version"`) && strings.Contains(bodyStr, `"dev"`) {
					fmt.Println("✓ App health check passed")
					return nil
				}
			} else {
				resp.Body.Close()
			}
		}
	}
}

func migrateDB(dbRelPath string) error {
	abs, err := filepath.Abs(dbRelPath)
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}
	cmd := exec.Command("./migrate.sh", "-p", "sqlite", "-u", abs)
	cmd.Dir = "../"
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("migrate %s: %w\n%s", dbRelPath, err, out)
	}
	return nil
}

func seedDB() error {
	db, err := sql.Open("sqlite", "file:../test-db.sqlite3")
	if err != nil {
		return err
	}
	defer db.Close()
	b, err := os.ReadFile("./testdata/seed.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(b))
	if err != nil {
		return err
	}
	return nil
}

// chromiumBlockedPorts lists ports that Chromium refuses to connect to.
// See https://fetch.spec.whatwg.org/#bad-port
var chromiumBlockedPorts = map[int]bool{
	3659: true, 4045: true, 4190: true, 4444: true, 4445: true,
	4782: true, 4783: true, 6000: true, 6001: true, 6002: true,
	6003: true, 6004: true, 6005: true, 6006: true, 6007: true,
	6008: true, 6009: true, 6010: true, 6011: true, 6012: true,
	6013: true, 6014: true, 6015: true, 6016: true, 6017: true,
	6018: true, 6019: true, 6020: true, 6021: true, 6022: true,
	6023: true, 6024: true, 6025: true, 6026: true, 6027: true,
	6028: true, 6029: true, 6030: true, 6031: true, 6032: true,
	6033: true, 6034: true, 6035: true, 6036: true, 6037: true,
	6038: true, 6039: true, 6040: true, 6041: true, 6042: true,
	6043: true, 6044: true, 6045: true, 6046: true, 6047: true,
	6048: true, 6049: true, 6050: true, 6051: true, 6052: true,
	6053: true, 6054: true, 6055: true, 6056: true, 6057: true,
	6058: true, 6059: true, 6060: true, 6061: true, 6062: true,
	6063: true, 6566: true, 6665: true, 6666: true, 6667: true,
	6668: true, 6669: true, 6679: true, 6697: true,
}

// getPort returns a free port that is not on Chromium's blocked-port list.
func getPort() int {
	for {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("could not find free port: %v", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		if !chromiumBlockedPorts[port] {
			return port
		}
	}
}

// afterAll does cleanup, e.g. stop playwright driver
func afterAll() {
	killAndWait(app)
	killAndWait(rateLimitApp)
	if err := pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
	if err := os.Remove("../test-db.sqlite3"); err != nil {
		log.Fatalf("could not remove test-db.sqlite3: %v", err)
	}
	if err := os.Remove("../test-rl-db.sqlite3"); err != nil {
		fmt.Println(err)
	}
	if err := os.Remove("../tmp/e2e-server"); err != nil {
		fmt.Println(err)
	}
}

// newPage creates a new browser context and page for each test,
// so each test has an isolated environment. Usage:
//
//	func TestFoo(t *testing.T) {
//	  _, page := newPage(t)
//	  // your test code
//	}
func newPage(t *testing.T, contextOptions ...playwright.BrowserNewContextOptions) (playwright.BrowserContext, playwright.Page) {
	t.Helper()
	opt := defaultContextOptions
	if len(contextOptions) == 1 {
		opt = contextOptions[0]
	}
	return newBrowserContextAndPage(t, opt)
}

func getBrowserName() string {
	browserName, hasEnv := os.LookupEnv("BROWSER")
	if hasEnv {
		return browserName
	}
	return "chromium"
}

func newBrowserContextAndPage(t *testing.T, options playwright.BrowserNewContextOptions) (playwright.BrowserContext, playwright.Page) {
	t.Helper()
	ctx, err := browser.NewContext(options)
	if err != nil {
		t.Fatalf("could not create new context: %v", err)
	}
	t.Cleanup(func() {
		if err := ctx.Close(); err != nil {
			t.Errorf("could not close context: %v", err)
		}
	})
	p, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("could not create new page: %v", err)
	}
	return ctx, p
}

func getFullPath(relativePath string) string {
	ref, err := url.Parse(relativePath)
	if err != nil {
		panic(err)
	}
	return baseURL.ResolveReference(ref).String()
}
