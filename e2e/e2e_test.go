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
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	_ "modernc.org/sqlite"
)

// global variables, can be used in any tests.
var (
	pw          *playwright.Playwright
	browser     playwright.Browser
	expect      playwright.PlaywrightAssertions
	isChromium  bool
	isFirefox   bool
	isWebKit    bool
	browserName = getBrowserName()
	browserType playwright.BrowserType
	app         *exec.Cmd
	appBinary   string
	baseURL     *url.URL
)

// defaultContextOptions for most tests
var defaultContextOptions = playwright.BrowserNewContextOptions{
	AcceptDownloads: playwright.Bool(true),
	HasTouch:        playwright.Bool(true),
}

func TestMain(m *testing.M) {
	beforeAll()
	code := m.Run()
	afterAll()
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
	// init web-first assertions with 3s timeout instead of default 5s
	expect = playwright.NewPlaywrightAssertions(3000)
	isChromium = browserName == "chromium" || browserName == ""
	isFirefox = browserName == "firefox"
	isWebKit = browserName == "webkit"

	if err = buildCSS(); err != nil {
		log.Fatalf("could not build CSS: %v", err)
	}

	if err = buildApp(); err != nil {
		log.Fatalf("could not build app: %v", err)
	}

	// start app.
	if err = startApp(); err != nil {
		log.Fatalf("could not start app: %v", err)
	}

	if err = waitForHealthCheck(baseURL.String()); err != nil {
		log.Fatalf("app failed health check: %v", err)
	}

	if err = seedDB(); err != nil {
		log.Fatalf("could not seed db: %v", err)
	}
}

func buildApp() error {
	f, err := os.CreateTemp("", "go-htmx-template-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	f.Close()
	appBinary = f.Name()

	cmd := exec.Command("go", "build", "-o", appBinary, "./cmd/server")
	cmd.Dir = "../"
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
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go-tw: %w\n%s", err, out)
	}
	return nil
}

func startApp() error {
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("getting free port: %w", err)
	}

	app = exec.Command(appBinary)
	app.Dir = "../"
	app.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	app.Env = append(
		os.Environ(),
		"DB_URL=./test-db.sqlite3",
		fmt.Sprintf("PORT=%d", port),
		"LOG_LEVEL=DEBUG",
	)

	baseURL, err = url.Parse(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return err
	}

	stdout, err := app.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := app.StderrPipe()
	if err != nil {
		return err
	}

	if err = app.Start(); err != nil {
		return err
	}
	fmt.Printf("Started app on port %d, pid %d", port, app.Process.Pid)

	stdoutchan := make(chan string)
	stderrchan := make(chan string)
	go func() {
		defer close(stdoutchan)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutchan <- scanner.Text()
		}
	}()
	go func() {
		defer close(stderrchan)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrchan <- scanner.Text()
		}
	}()

	go func() {
		for line := range stdoutchan {
			fmt.Println("[STDOUT]", line)
		}
	}()
	go func() {
		for line := range stderrchan {
			fmt.Println("[STDERR]", line)
		}
	}()
	return nil
}

// waitForHealthCheck polls the /health endpoint until it responds successfully.
// Returns nil on success, error on timeout or other failure.
func waitForHealthCheck(baseURL string) error {
	healthURL := baseURL + "/health"
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 500 * time.Millisecond, // Each request times out after 500ms
	}

	for {
		select {
		case <-timeout:
			return fmt.Errorf("health check timeout: app did not become ready within 10s")
		case <-ticker.C:
			resp, err := client.Get(healthURL)
			if err != nil {
				// Network error or app not ready, continue polling
				continue
			}

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}

				// Verify expected response (contains "version" field with JSON)
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

func getFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// afterAll does cleanup, e.g. stop playwright driver
func afterAll() {
	if app != nil && app.Process != nil {
		if err := syscall.Kill(-app.Process.Pid, syscall.SIGKILL); err != nil {
			fmt.Println(err)
		}
	}
	if appBinary != "" {
		os.Remove(appBinary) //nolint:errcheck
	}
	if err := pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
	if err := os.Remove("../test-db.sqlite3"); err != nil {
		log.Fatalf("could not remove test-db.sqlite3: %v", err)
	}
}

// beforeEach creates a new browser context and page for each test,
// so each test has an isolated environment. Usage:
//
//	func TestFoo(t *testing.T) {
//	  _, page := beforeEach(t)
//	  // your test code
//	}
func beforeEach(t *testing.T, contextOptions ...playwright.BrowserNewContextOptions) (playwright.BrowserContext, playwright.Page) {
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
		ref = &url.URL{Path: relativePath}
	}
	return baseURL.ResolveReference(ref).String()
}
