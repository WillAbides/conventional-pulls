package conventionalpulls

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

//go:generate mockgen -source $GOFILE -destination internal/mocks/mock_$GOFILE -package mocks

var defaultLabelValues = map[string]VersionChange{
	"Non-Production Change": VersionChangeNone,
	"Patch":                 VersionChangePatch,
	"Minor Change":          VersionChangeMinor,
	"Breaking Change":       VersionChangeMajor,
}

// VersionChange represents the amount to increment a semver
type VersionChange int

// All VersionChanges
const (
	VersionChangeNone VersionChange = iota
	VersionChangePatch
	VersionChangeMinor
	VersionChangeMajor
	versionChangeInvalid // this will be at the bottom of the list so we can range over VersionChanges
)

var versionChangeNames = map[VersionChange]string{
	VersionChangeNone:    "None",
	VersionChangePatch:   "Patch",
	VersionChangeMinor:   "Minor",
	VersionChangeMajor:   "Major",
	versionChangeInvalid: "Invalid",
}

func (vc VersionChange) valid() bool {
	return vc >= 0 && vc < versionChangeInvalid
}

func (vc VersionChange) mustBeValid() {
	if !vc.valid() {
		panic(fmt.Sprintf("%d is not a valid VersionChange", vc))
	}
}

func (vc VersionChange) String() string {
	if !vc.valid() {
		return versionChangeNames[versionChangeInvalid]
	}
	return versionChangeNames[vc]
}

// greater returns whichever is higher, ch or other. Panics if either value is invalid.
func (vc VersionChange) greater(other VersionChange) VersionChange {
	vc.mustBeValid()
	other.mustBeValid()
	if other > vc {
		return other
	}
	return vc
}

// PRLabelFetcher fetches PR labels from GitHub (or wherever)
type PRLabelFetcher interface {
	FetchPRLabels(id int) (labels []string, err error)
}

// Config configuration values
type Config struct {
	LabelValues    map[string]VersionChange
	RequireLabels  bool
	PRLabelFetcher PRLabelFetcher
}

func (cfg *Config) prLabels(prIDs []int) (map[int][]string, error) {
	if cfg.PRLabelFetcher == nil {
		panic("PRLabelFetcher shant be nil")
	}
	result := make(map[int][]string, len(prIDs))
	for _, id := range prIDs {
		labels, err := cfg.PRLabelFetcher.FetchPRLabels(id)
		if err != nil {
			return nil, &PRLabelFetcherErr{err: err}
		}
		result[id] = make([]string, len(labels))
		for i, label := range labels {
			result[id][i] = strings.ToLower(label)
		}
	}
	return result, nil
}

// PRVersionChange what level of change is required for the given pulls
func (cfg *Config) PRVersionChange(pullRequestID ...int) (VersionChange, error) {
	versionChange := VersionChangeNone
	prLabels, err := cfg.prLabels(pullRequestID)
	if err != nil {
		return 0, err
	}
	err = cfg.requireLabels(prLabels)
	if err != nil {
		return 0, err
	}
	for _, labels := range prLabels {
		versionChange = cfg.maxVersionChange(labels).greater(versionChange)
	}
	return versionChange, nil
}

// NextVersion returns the next version for a release including the given pulls
func (cfg *Config) NextVersion(prevVersion string, pullRequestID ...int) (string, error) {
	bump, err := cfg.PRVersionChange(pullRequestID...)
	if err != nil {
		return "", err
	}
	return nextVersion(prevVersion, bump)
}

func nextVersion(previousVersion string, bump VersionChange) (string, error) {
	bump.mustBeValid()
	prev, err := semver.NewVersion(previousVersion)
	if err != nil {
		return "", fmt.Errorf("could not parse semver from %q", previousVersion)
	}
	var next semver.Version
	switch bump {
	case VersionChangeNone:
		next = *prev
	case VersionChangePatch:
		next = prev.IncPatch()
	case VersionChangeMinor:
		next = prev.IncMinor()
	case VersionChangeMajor:
		next = prev.IncMajor()
	}
	return next.Original(), nil
}

func (cfg *Config) requireLabels(prLabels map[int][]string) error {
	if !cfg.RequireLabels {
		return nil
	}
	prIDs := make([]int, 0, len(prLabels))
	for id := range prLabels {
		prIDs = append(prIDs, id)
	}
	sort.Ints(prIDs)
	var err PRMissingLabelErr
	for _, id := range prIDs {
		labels := prLabels[id]
		if !cfg.containsAnyLabel(labels) {
			err.IDs = append(err.IDs, id)
		}
	}
	if len(err.IDs) > 0 {
		return &err
	}
	return nil
}

func (cfg *Config) labelValues() map[string]VersionChange {
	labels := defaultLabelValues
	if cfg.LabelValues != nil {
		labels = cfg.LabelValues
	}
	result := make(map[string]VersionChange, len(labels))
	for k, v := range labels {
		result[strings.ToLower(k)] = v
	}
	return result
}

// containsAnyLabel returns true if any of LabelValues is part of cfg.LabelValues.
func (cfg *Config) containsAnyLabel(labels []string) bool {
	labelValues := cfg.labelValues()
	for _, label := range labels {
		_, ok := labelValues[strings.ToLower(label)]
		if ok {
			return true
		}
	}
	return false
}

// maxVersionChange returns the maximum version change configured for any of the given LabelValues.
// Returns VersionChangeNone if none have any configured change.
func (cfg *Config) maxVersionChange(labels []string) VersionChange {
	change := VersionChangeNone
	labelValues := cfg.labelValues()
	for _, label := range labels {
		labelChange := labelValues[strings.ToLower(label)]
		change = labelChange.greater(change)
	}
	return change
}

// PRMissingLabelErr is an error indicating that one or more pull requests aren't properly labeled.
type PRMissingLabelErr struct {
	IDs []int
}

func (e *PRMissingLabelErr) Error() string {
	return "one or more PRs have no configured labels"
}

// PRLabelFetcherErr is an error indicating a problem fetching pull request labels.
type PRLabelFetcherErr struct {
	err error
}

// Unwrap meets xerrors.Wrapper
func (e *PRLabelFetcherErr) Unwrap() error {
	return e.err
}

func (e *PRLabelFetcherErr) Error() string {
	return "error from PRLabelFetcher"
}
