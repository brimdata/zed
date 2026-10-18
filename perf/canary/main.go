// Command canary compares the performance of two builds of super by
// loading and querying the small "ghasample" data set with each build and
// reporting significant differences in load time, query time, and memory.
// See README.md for details.
package main

import (
	"cmp"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	jsonURL    = "https://brim-benchmarks.s3.us-east-2.amazonaws.com/gha/ghasample.json.gz"
	parquetURL = "https://brim-benchmarks.s3.us-east-2.amazonaws.com/gha/ghasample.parquet"
)

//go:embed queries
var queryFS embed.FS

// The queries in queries/, in the order they're run and reported.
var queryFiles = []string{"search.spq", "search+.spq", "count.sql", "agg.sql", "union.spq"}

// Each system runs the queries on one input format.
type system struct {
	name  string
	flags []string
	input func(b *build, data string) string
}

var systems = []system{
	{"super-bsup", []string{"-i", "bsup", "-f", "csv"}, func(b *build, _ string) string { return b.bsup }},
	{"super-json", []string{"-dynamic", "-i", "json", "-f", "csv"}, func(_ *build, data string) string { return filepath.Join(data, "ghasample.json.gz") }},
	{"super-json-static", []string{"-static", "-i", "json", "-f", "csv"}, func(_ *build, data string) string { return filepath.Join(data, "ghasample.json.gz") }},
	{"super-parquet", []string{"-i", "parquet", "-f", "csv"}, func(_ *build, data string) string { return filepath.Join(data, "ghasample.parquet") }},
}

type build struct {
	Label   string `json:"label"`
	Rev     string `json:"rev"`     // commit hash, or "" for the working tree
	Version string `json:"version"` // git describe output
	// Merged is the git describe output for the second parent when the
	// target is a merge commit whose first parent is the base, as it is
	// for a pull request run in GitHub Actions.
	Merged string `json:"merged,omitempty"`
	commit string // commit hash of a clean working tree
	bin    string
	bsup   string
}

type measurement struct {
	Wall   float64 `json:"wall"`   // seconds
	User   float64 `json:"user"`   // seconds
	Sys    float64 `json:"sys"`    // seconds
	MaxRSS int64   `json:"maxrss"` // bytes (0 if unavailable)
}

func (m measurement) cpu() float64 { return m.User + m.Sys }

type result struct {
	Test           string        `json:"test"`
	Base           []measurement `json:"base"`
	Target         []measurement `json:"target"`
	BaseErr        string        `json:"base_error,omitempty"`
	TargetErr      string        `json:"target_error,omitempty"`
	OutputMismatch bool          `json:"output_mismatch,omitempty"`
}

type config struct {
	base, target, mainRef string
	pr                    string
	runs                  int
	timeout               time.Duration
	threshold             float64
	minDelta              time.Duration
	minMemDelta           int64
	dataDir, outDir       string
	keep                  bool
	systems               []system
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("perf-canary: ")
	var c config
	cacheDir, _ := os.UserCacheDir()
	flag.StringVar(&c.base, "base", "", "base commit (default: merge-base of target and -main)")
	flag.StringVar(&c.target, "target", "", "target commit (default: the working tree, including uncommitted changes)")
	flag.StringVar(&c.mainRef, "main", "origin/main", "ref used to compute the default base")
	flag.StringVar(&c.pr, "pr", "", "URL of the pull request being tested, for the report")
	flag.IntVar(&c.runs, "runs", 5, "measured runs per query for each build")
	flag.DurationVar(&c.timeout, "timeout", 5*time.Minute, "timeout for each super invocation")
	flag.Float64Var(&c.threshold, "threshold", 0.25, "relative change that counts as significant")
	flag.DurationVar(&c.minDelta, "min-delta", 25*time.Millisecond, "minimum absolute time change that counts as significant")
	minMemMB := flag.Int64("min-mem-delta", 20, "minimum absolute peak memory change in MB that counts as significant")
	flag.StringVar(&c.dataDir, "datadir", filepath.Join(cacheDir, "super-perf-canary"), "directory for cached test data")
	flag.StringVar(&c.outDir, "out", "", "directory for summary.md, results.json, and query outputs (default: a new temporary directory, which is kept)")
	flag.BoolVar(&c.keep, "keep", false, "keep the temporary work directory")
	rereportPath := flag.String("rereport", "", "reclassify the results in this results.json file instead of running")
	systemNames := flag.String("systems", allSystemNames(), "comma-separated list of systems to test")
	// The default usage line names the temporary binary built by "go run".
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "Usage: go run ./perf/canary [flags]\n\n"+
			"Compares the performance of two builds of super. See perf/canary/README.md.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	c.minMemDelta = *minMemMB << 20
	var err error
	if c.systems, err = selectSystems(*systemNames); err != nil {
		log.Fatal(err)
	}
	if *rereportPath != "" {
		rep, err := rereport(c, *rereportPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(rep.markdown())
		return
	}
	if err := run(c); err != nil {
		log.Fatal(err)
	}
}

func run(c config) error {
	repo, err := git("", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "super-perf-canary-")
	if err != nil {
		return err
	}
	if c.keep {
		log.Printf("work directory: %s", work)
	} else {
		defer os.RemoveAll(work)
	}
	if c.outDir == "" {
		// Not in the work directory, so the results outlive it.
		if c.outDir, err = os.MkdirTemp("", "super-perf-canary-results-"); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(c.outDir, "outputs"), 0755); err != nil {
		return err
	}
	base, target, err := resolveBuilds(repo, c)
	if err != nil {
		return err
	}
	if err := fetchData(c.dataDir); err != nil {
		return err
	}
	for _, b := range []*build{base, target} {
		if err := buildSuper(repo, work, b); err != nil {
			return err
		}
	}
	var results []*result
	// The BSUP "load" is measured once per build.
	load := &result{Test: "super-bsup load"}
	for _, b := range []*build{base, target} {
		if !c.has("super-bsup") {
			break
		}
		b.bsup = filepath.Join(work, b.Label+".bsup")
		log.Printf("loading BSUP with %s", b.Label)
		m, _, err := measure(c.timeout, b.bin, "-dynamic", "-i", "json", "-f", "bsup", "-o", b.bsup, filepath.Join(c.dataDir, "ghasample.json.gz"))
		if err != nil {
			return fmt.Errorf("BSUP load with %s build: %w", b.Label, err)
		}
		if b == base {
			load.Base = append(load.Base, m)
		} else {
			load.Target = append(load.Target, m)
		}
	}
	if c.has("super-bsup") {
		results = append(results, load)
	}
	for _, sys := range c.systems {
		for _, qf := range queryFiles {
			r, err := runQuery(c, work, sys, qf, base, target)
			if err != nil {
				return err
			}
			results = append(results, r)
		}
	}
	rep := newReport(c, base, target, results)
	if err := rep.write(c.outDir); err != nil {
		return err
	}
	fmt.Print(rep.markdown())
	log.Printf("report written to %s", filepath.Join(c.outDir, "summary.md"))
	log.Printf("raw results and query outputs are in %s", c.outDir)
	return nil
}

func allSystemNames() string {
	return strings.Join(systemNamesOf(systems), ",")
}

func selectSystems(list string) ([]system, error) {
	var selected []system
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		i := slices.IndexFunc(systems, func(sys system) bool { return sys.name == name })
		if i < 0 {
			return nil, fmt.Errorf("unknown system %q (choose from %s)", name, allSystemNames())
		}
		if !slices.ContainsFunc(selected, func(sys system) bool { return sys.name == name }) {
			selected = append(selected, systems[i])
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no systems selected")
	}
	return selected, nil
}

func (c config) has(name string) bool {
	return slices.ContainsFunc(c.systems, func(sys system) bool { return sys.name == name })
}

func resolveBuilds(repo string, c config) (*build, *build, error) {
	target := &build{Label: "target"}
	targetRef := "HEAD"
	if c.target != "" {
		rev, err := git(repo, "rev-parse", "--verify", c.target+"^{commit}")
		if err != nil {
			return nil, nil, err
		}
		target.Rev = rev
		targetRef = rev
		target.Version, _ = git(repo, "describe", "--tags", "--always", rev)
	} else if dirty, _ := git(repo, "status", "--porcelain", "--untracked-files=no"); dirty != "" {
		target.Version, _ = git(repo, "describe", "--tags", "--always", "--dirty")
		target.Version += " (working tree)"
	} else {
		// A clean working tree builds the same as HEAD.
		target.Version, _ = git(repo, "describe", "--tags", "--always")
		target.commit, _ = git(repo, "rev-parse", "HEAD")
	}
	base := &build{Label: "base"}
	baseRef := c.base
	if baseRef == "" {
		mb, err := git(repo, "merge-base", targetRef, c.mainRef)
		if err != nil {
			return nil, nil, fmt.Errorf("computing merge-base with %s (try -base or -main): %w", c.mainRef, err)
		}
		baseRef = mb
	}
	rev, err := git(repo, "rev-parse", "--verify", baseRef+"^{commit}")
	if err != nil {
		return nil, nil, err
	}
	base.Rev = rev
	base.Version, _ = git(repo, "describe", "--tags", "--always", rev)
	if commit := cmp.Or(target.Rev, target.commit); commit != "" {
		// The output is the commit followed by its parents.
		out, _ := git(repo, "rev-list", "--parents", "-n", "1", commit)
		if f := strings.Fields(out); len(f) == 3 && f[1] == base.Rev {
			target.Merged, _ = git(repo, "describe", "--tags", "--always", f[2])
		}
	}
	return base, target, nil
}

func fetchData(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, u := range []string{jsonURL, parquetURL} {
		path := filepath.Join(dir, filepath.Base(u))
		if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
			continue
		}
		log.Printf("downloading %s", u)
		if err := download(u, path); err != nil {
			return err
		}
	}
	return nil
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && resp.ContentLength >= 0 && n != resp.ContentLength {
		err = fmt.Errorf("%s: got %d bytes, expected %d", url, n, resp.ContentLength)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func buildSuper(repo, work string, b *build) error {
	b.bin = filepath.Join(work, "bin", "super-"+b.Label)
	dir := repo
	if b.Rev != "" {
		dir = filepath.Join(work, "src-"+b.Label)
		if _, err := git(repo, "worktree", "add", "--detach", dir, b.Rev); err != nil {
			return err
		}
		defer git(repo, "worktree", "remove", "--force", dir)
	}
	log.Printf("building %s: %s", b.Label, b.Version)
	ldflags := "-s -X github.com/superdb/super/cli.version=" + strings.Fields(b.Version)[0]
	cmd := exec.Command("go", "build", "-ldflags="+ldflags, "-o", b.bin, "./cmd/super")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building %s: %w", b.Label, err)
	}
	return nil
}

func runQuery(c config, work string, sys system, qf string, base, target *build) (*result, error) {
	name := strings.TrimSuffix(qf, filepath.Ext(qf))
	r := &result{Test: sys.name + " " + name}
	text, err := queryFS.ReadFile("queries/" + qf)
	if err != nil {
		return nil, err
	}
	args := map[*build][]string{}
	for _, b := range []*build{base, target} {
		path := filepath.Join(work, fmt.Sprintf("%s-%s-%s", sys.name, name, b.Label)+filepath.Ext(qf))
		q := strings.ReplaceAll(string(text), "__SOURCE__", sys.input(b, c.dataDir))
		if err := os.WriteFile(path, []byte(q), 0644); err != nil {
			return nil, err
		}
		args[b] = append(append([]string{}, sys.flags...), "-I", path)
	}
	log.Printf("running %s", r.Test)
	// The warmup run for each build also provides the output to compare.
	outputs := map[*build][]byte{}
	failed := map[*build]bool{}
	fail := func(b *build, err error) {
		failed[b] = true
		if b == base {
			r.BaseErr = err.Error()
		} else {
			r.TargetErr = err.Error()
		}
	}
	for _, b := range []*build{base, target} {
		_, out, err := measure(c.timeout, b.bin, args[b]...)
		if err != nil {
			fail(b, err)
			continue
		}
		outputs[b] = out
		file := filepath.Join(c.outDir, "outputs", fmt.Sprintf("%s-%s.%s.csv", sys.name, name, b.Label))
		if err := os.WriteFile(file, out, 0644); err != nil {
			return nil, err
		}
	}
	if !failed[base] && !failed[target] && string(outputs[base]) != string(outputs[target]) {
		r.OutputMismatch = true
	}
	// Alternate the order of the builds (ABBA...) so any drift in the
	// machine's performance affects both builds equally.
	for i := 0; i < c.runs; i++ {
		order := []*build{base, target}
		if i%2 == 1 {
			order = []*build{target, base}
		}
		for _, b := range order {
			if failed[b] {
				continue
			}
			m, _, err := measure(c.timeout, b.bin, args[b]...)
			if err != nil {
				fail(b, err)
				continue
			}
			if b == base {
				r.Base = append(r.Base, m)
			} else {
				r.Target = append(r.Target, m)
			}
		}
	}
	return r, nil
}

// measure runs a command and returns its wall time, CPU time, peak memory,
// and standard output.
func measure(timeout time.Duration, name string, args ...string) (measurement, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	start := time.Now()
	out, err := cmd.Output()
	wall := time.Since(start)
	if ctx.Err() != nil {
		return measurement{}, nil, fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return measurement{}, nil, errors.New(firstLine(msg))
	}
	ps := cmd.ProcessState
	return measurement{
		Wall:   wall.Seconds(),
		User:   ps.UserTime().Seconds(),
		Sys:    ps.SystemTime().Seconds(),
		MaxRSS: maxRSS(ps),
	}, out, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func median(ms []measurement, f func(measurement) float64) float64 {
	if len(ms) == 0 {
		return 0
	}
	vals := make([]float64, len(ms))
	for i, m := range ms {
		vals[i] = f(m)
	}
	sort.Float64s(vals)
	n := len(vals)
	if n%2 == 1 {
		return vals[n/2]
	}
	return (vals[n/2-1] + vals[n/2]) / 2
}

func hostInfo() string {
	return fmt.Sprintf("%s/%s, %d CPUs%s", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), cpuModel())
}

func cpuModel() string {
	var out []byte
	switch runtime.GOOS {
	case "linux":
		b, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(b), "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "model name" {
				return ", " + strings.TrimSpace(v)
			}
		}
		return ""
	case "darwin":
		out, _ = exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	}
	if s := strings.TrimSpace(string(out)); s != "" {
		return ", " + s
	}
	return ""
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func writeFile(path, s string) error {
	return os.WriteFile(path, []byte(s), 0644)
}
