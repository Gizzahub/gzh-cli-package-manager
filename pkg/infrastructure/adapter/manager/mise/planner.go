package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	adapterm "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager"
	"github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

type upgradePlan struct{ Name, Target string }

func (a *Adapter) plan(ctx context.Context, opts adapterm.UpdateOptions, tools []miseTool) ([]upgradePlan, error) {
	seen := map[string]bool{}
	plans := make([]upgradePlan, 0, len(tools))
	for _, tool := range tools {
		if seen[tool.Name] {
			return nil, fmt.Errorf("mise update: %s has multiple active versions; policy update is unsafe", tool.Name)
		}
		seen[tool.Name] = true
		plan, err := a.planTool(ctx, opts, tool)
		if err != nil {
			return nil, err
		}
		if plan != nil {
			plans = append(plans, *plan)
		}
	}
	return plans, nil
}

func (a *Adapter) planTool(ctx context.Context, opts adapterm.UpdateOptions, tool miseTool) (*upgradePlan, error) {
	if len(opts.Packages) != 0 && !contains(opts.Packages, tool.Name) {
		return nil, nil
	}
	installed, ok := parseSemver(tool.Version)
	if !ok {
		return nil, fmt.Errorf("mise update: %s installed version %q is not numeric semver", tool.Name, tool.Version)
	}
	req, wildcard, err := parseRequest(tool.Requested)
	if err != nil {
		return nil, fmt.Errorf("mise update: %s: %w", tool.Name, err)
	}
	if !opts.Bump && !wildcard && req.parts == 3 {
		return nil, nil
	}
	versions, err := a.remote(ctx, opts, tool.Name, opts.Strategy == adapterm.StrategyLatest)
	if err != nil {
		return nil, err
	}
	best := selectVersion(versions, installed, req, wildcard, opts)
	if best == nil {
		return nil, nil
	}
	return &upgradePlan{Name: tool.Name, Target: best.String()}, nil
}

func selectVersion(versions []semver, installed semver, req request, wildcard bool, opts adapterm.UpdateOptions) *semver {
	var best *semver
	for i := range versions {
		candidate := &versions[i]
		if eligible(*candidate, installed, req, wildcard, opts) && (best == nil || candidate.compare(*best) > 0) {
			best = candidate
		}
	}
	return best
}

func eligible(candidate, installed semver, req request, wildcard bool, opts adapterm.UpdateOptions) bool {
	if candidate.compare(installed) <= 0 || !allowedPrerelease(candidate, opts) {
		return false
	}
	if !withinStrategy(candidate, installed, opts.Strategy) {
		return false
	}
	return opts.Bump || wildcard || req.matches(candidate)
}

func allowedPrerelease(candidate semver, opts adapterm.UpdateOptions) bool {
	return candidate.pre == "" || (opts.Strategy == adapterm.StrategyLatest && opts.Bump)
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (a *Adapter) remote(ctx context.Context, opts adapterm.UpdateOptions, name string, prerelease bool) ([]semver, error) {
	remoteOpts := opts
	remoteOpts.MiseLocal = false // remote catalog lookup has no local configuration scope.
	args := a.args(remoteOpts, "ls-remote", name, "--json")
	if prerelease {
		args = append(args, "--prerelease")
	}
	r, execErr := a.executor.Execute(ctx, command, args...)
	if err := cmdutil.CheckResult(r, execErr, "list remote mise versions for "+name); err != nil {
		return nil, err
	}
	versions, err := decodeVersions(r.Stdout)
	if err != nil {
		return nil, fmt.Errorf("parse remote mise versions for %s: %w", name, err)
	}
	return versions, nil
}

func decodeVersions(data string) ([]semver, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, fmt.Errorf("decode versions array: %w", err)
	}
	versions := make([]semver, 0, len(raw))
	for _, value := range raw {
		var text string
		if json.Unmarshal(value, &text) != nil {
			var object struct {
				Version string `json:"version"`
			}
			if err := json.Unmarshal(value, &object); err != nil {
				return nil, fmt.Errorf("decode version object: %w", err)
			}
			text = object.Version
		}
		if parsed, ok := parseSemver(text); ok {
			versions = append(versions, parsed)
		}
	}
	return versions, nil
}

type request struct {
	major, minor, patch int
	parts               int
}

func parseRequest(value string) (request, bool, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if value == "" {
		return request{}, false, fmt.Errorf("missing requested version")
	}
	if value == "*" || value == "x" || value == "latest" {
		return request{}, true, nil
	}
	if strings.ContainsAny(value, "<>=~^| ") || strings.Contains(value, "-") {
		return request{}, false, fmt.Errorf("requested version %q is a symbolic/channel range", value)
	}
	parts := strings.Split(value, ".")
	if len(parts) > 3 {
		return request{}, false, fmt.Errorf("requested version %q is not a numeric semver prefix", value)
	}
	parsed := request{parts: len(parts)}
	values := []*int{&parsed.major, &parsed.minor, &parsed.patch}
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return request{}, false, fmt.Errorf("requested version %q is not numeric", value)
		}
		*values[i] = number
	}
	return parsed, false, nil
}

func (r request) matches(v semver) bool {
	if r.parts >= 1 && r.major != v.major {
		return false
	}
	if r.parts >= 2 && r.minor != v.minor {
		return false
	}
	if r.parts < 3 {
		return true
	}
	return r.patch == v.patch && v.pre == ""
}

func withinStrategy(v, installed semver, strategy adapterm.UpdateStrategy) bool {
	switch strategy {
	case adapterm.StrategyMinor:
		return v.major == installed.major
	case adapterm.StrategyMicro:
		return v.major == installed.major && v.minor == installed.minor
	case adapterm.StrategyLatest:
		return true
	default:
		return false
	}
}

type semver struct {
	major, minor, patch int
	pre                 string
	raw                 string
}

func parseSemver(value string) (semver, bool) {
	raw := strings.TrimSpace(value)
	value = strings.TrimPrefix(raw, "v")
	coreAndBuild := strings.SplitN(value, "+", 2)
	if len(coreAndBuild) == 2 && !validIdentifiers(coreAndBuild[1]) {
		return semver{}, false
	}
	parts := strings.SplitN(coreAndBuild[0], "-", 2)
	numbers := strings.Split(parts[0], ".")
	if len(numbers) != 3 {
		return semver{}, false
	}
	parsed := semver{raw: raw}
	values := []*int{&parsed.major, &parsed.minor, &parsed.patch}
	for i, number := range numbers {
		value, err := strconv.Atoi(number)
		if err != nil || value < 0 || (len(number) > 1 && number[0] == '0') {
			return semver{}, false
		}
		*values[i] = value
	}
	if len(parts) == 2 {
		parsed.pre = parts[1]
		if parsed.pre == "" || !validIdentifiers(parsed.pre) {
			return semver{}, false
		}
	}
	return parsed, true
}

func (v semver) String() string {
	return v.raw
}

func (v semver) compare(other semver) int {
	left, right := []int{v.major, v.minor, v.patch}, []int{other.major, other.minor, other.patch}
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return comparePrerelease(v.pre, other.pre)
}
