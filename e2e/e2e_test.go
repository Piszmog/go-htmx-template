//go:build e2e

package e2e_test

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// global variables, can be used in any tests
var (
	pw          *playwright.Playwright
	browser     playwright.Browser
	context     playwright.BrowserContext
	page        playwright.Page
	expect      playwright.PlaywrightAssertions
	isChromium  bool
	isFirefox   bool
	isWebKit    bool
	browserName = getBrowserName()
	browserType playwright.BrowserType
	app         *exec.Cmd
	baseUrL     *url.URL
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
	// init web-first assertions with 1s timeout instead of default 5s
	expect = playwright.NewPlaywrightAssertions(1000)
	isChromium = browserName == "chromium" || browserName == ""
	isFirefox = browserName == "firefox"
	isWebKit = browserName == "webkit"

	// start app
	if err = startApp(); err != nil {
		log.Fatalf("could not start app: %v", err)
	}
	time.Sleep(time.Second * 5)
	if err = seedDB(); err != nil {
		log.Fatalf("could not seed db: %v", err)
	}
}

func startApp() error {
	port := getPort()
	app = exec.Command("go", "run", "./cmd/server")
	app.Dir = "../"
	app.Env = append(
		os.Environ(),
		"DB_URL=./test-db.sqlite3",
		fmt.Sprintf("PORT=%d", port),
		"LOG_LEVEL=DEBUG",
	)

	var err error
	baseUrL, err = url.Parse(fmt.Sprintf("http://localhost:%d", port))
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

	if err := app.Start(); err != nil {
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

func seedDB() error {
	db, err := sql.Open("libsql", "file:../test-db.sqlite3")
	if err != nil {
		return err
	}
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

func getPort() int {
	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	return randomGenerator.Intn(9001-3000) + 3000
}

// afterAll does cleanup, e.g. stop playwright driver
func afterAll() {
	if app != nil && app.Process != nil {
		if err := syscall.Kill(-app.Process.Pid, syscall.SIGKILL); err != nil {
			fmt.Println(err)
		}
	}
	if err := pw.Stop(); err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	if err := os.Remove("../test-db.sqlite3"); err != nil {
		log.Fatalf("could not remove test-db.sqlite3: %v", err)
	}
}

// beforeEach creates a new context and page for each test,
// so each test has isolated environment. Usage:
//
//	Func TestFoo(t *testing.T) {
//	  beforeEach(t)
//	  // your test code
//	}
func beforeEach(t *testing.T, contextOptions ...playwright.BrowserNewContextOptions) {
	t.Helper()
	opt := defaultContextOptions
	if len(contextOptions) == 1 {
		opt = contextOptions[0]
	}
	context, page = newBrowserContextAndPage(t, opt)
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
	context, err := browser.NewContext(options)
	if err != nil {
		t.Fatalf("could not create new context: %v", err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Errorf("could not close context: %v", err)
		}
	})
	p, err := context.NewPage()
	if err != nil {
		t.Fatalf("could not create new page: %v", err)
	}
	return context, p
}

func getFullPath(relativePath string) string {
	return baseUrL.ResolveReference(&url.URL{Path: relativePath}).String()
}
