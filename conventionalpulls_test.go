package conventionalpulls

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/conventionalpulls/internal/mocks"
)

func TestVersionChange_String(t *testing.T) {
	t.Run("versionChangeInvalid.String()", func(t *testing.T) {
		require.NotEmpty(t, versionChangeInvalid.String())
	})
	t.Run("all have a string", func(t *testing.T) {
		for change := VersionChange(0); change < versionChangeInvalid; change++ {
			require.NotEmpty(t, change.String())
			require.NotEqual(t, versionChangeInvalid.String(), change.String())
		}
	})
	t.Run("negative is invalid", func(t *testing.T) {
		require.Equal(t, versionChangeInvalid.String(), VersionChange(-1).String())
	})
	t.Run("versionChangeInvalid + 1 is invalid", func(t *testing.T) {
		change := versionChangeInvalid + 1
		require.Equal(t, versionChangeInvalid.String(), change.String())
	})
}

func TestVersionChange_valid(t *testing.T) {
	require.True(t, VersionChangeMajor.valid())
	require.True(t, VersionChangeNone.valid())
	require.False(t, VersionChange(-1).valid())
	require.False(t, versionChangeInvalid.valid())
	require.False(t, (versionChangeInvalid + 1).valid())
}

func TestVersionChangeGreater(t *testing.T) {
	require.Panics(t, func() {
		VersionChangeMajor.greater(versionChangeInvalid)
	})
	require.Panics(t, func() {
		versionChangeInvalid.greater(VersionChangeMajor)
	})
	require.Equal(t, VersionChangeMajor, VersionChangeMajor.greater(VersionChangeMajor))
	require.Equal(t, VersionChangeMajor, VersionChangeMajor.greater(VersionChangeMinor))
	require.Equal(t, VersionChangeMajor, VersionChangeMinor.greater(VersionChangeMajor))
}

func TestConfig_prLabels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return(nil, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
		}
		ids := []int{1, 2, 3}
		want := map[int][]string{
			1: {"foo", "bar"},
			2: {"baz", "qux"},
			3: {},
		}
		got, err := cfg.prLabels(ids)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX"}, assert.AnError)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
		}
		got, err := cfg.prLabels([]int{1, 2, 3})
		require.Error(t, err)
		require.Equal(t, &PRLabelFetcherErr{
			err: assert.AnError,
		}, err)
		require.Nil(t, got)
	})
}

func Test_clConfig_containsAnyLabel(t *testing.T) {
	t.Run("nil cfg.LabelValues", func(t *testing.T) {
		cfg := new(Config)
		require.False(t, cfg.containsAnyLabel([]string{"A"}))
	})

	t.Run("has a hit", func(t *testing.T) {
		cfg := &Config{
			LabelValues: map[string]VersionChange{
				"b": VersionChangeNone,
			},
		}
		require.True(t, cfg.containsAnyLabel([]string{"A", "B", "C"}))
	})

	t.Run("no hit", func(t *testing.T) {
		cfg := &Config{
			LabelValues: map[string]VersionChange{
				"b": VersionChangeNone,
			},
		}
		require.False(t, cfg.containsAnyLabel([]string{"A", "C"}))
	})
}

func Test_clConfig_maxVersionChange(t *testing.T) {
	t.Run("nil cfg.LabelValues", func(t *testing.T) {
		cfg := new(Config)
		require.Equal(t, VersionChangeNone, cfg.maxVersionChange([]string{"A"}))
	})

	t.Run("major", func(t *testing.T) {
		cfg := &Config{
			LabelValues: map[string]VersionChange{
				"a": VersionChangeNone,
				"b": VersionChangeMajor,
				"c": VersionChangeMinor,
			},
		}
		require.Equal(t, VersionChangeMajor, cfg.maxVersionChange([]string{"A", "B", "C", "D"}))
	})

	t.Run("no hit", func(t *testing.T) {
		cfg := &Config{
			LabelValues: map[string]VersionChange{
				"b": VersionChangePatch,
			},
		}
		require.Equal(t, VersionChangeNone, cfg.maxVersionChange([]string{"A", "C", "D"}))
	})
}

func TestPRLabelFetcherErr(t *testing.T) {
	err := &PRLabelFetcherErr{
		err: assert.AnError,
	}
	require.Equal(t, "error from PRLabelFetcher", err.Error())
	require.EqualError(t, err.Unwrap(), assert.AnError.Error())
}

func TestConfig_PRVersionChange(t *testing.T) {
	t.Run("no labels required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "minor change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return(nil, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  false,
		}
		got, err := cfg.PRVersionChange(1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, VersionChangeMinor, got)
	})

	t.Run("has labels required", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "minor change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		got, err := cfg.PRVersionChange(1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, VersionChangeMinor, got)
	})

	t.Run("missing required labels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "minor change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return(nil, nil)
		mockFetcher.EXPECT().FetchPRLabels(4).Return([]string{"a"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		wantErr := &PRMissingLabelErr{
			IDs: []int{3, 4},
		}
		got, err := cfg.PRVersionChange(3, 1, 4, 2)
		require.EqualError(t, err, "one or more PRs have no configured labels")
		require.Equal(t, wantErr, err)
		require.Equal(t, VersionChangeNone, got)
	})

	t.Run("fetcher error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "minor change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return(nil, assert.AnError)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  false,
		}
		got, err := cfg.PRVersionChange(1, 2, 3)
		require.Error(t, err)
		require.Equal(t, VersionChangeNone, got)
	})

	t.Run("nil fetcher", func(t *testing.T) {
		cfg := new(Config)
		require.Panics(t, func() {
			_, err := cfg.PRVersionChange(1, 2, 3)
			require.NoError(t, err)
		})
	})
}

func TestConfig_NextVersion(t *testing.T) {
	t.Run("no change", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "non-production change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "non-production change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		got, err := cfg.NextVersion("v1.2.3", 1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, "v1.2.3", got)
	})

	t.Run("patch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "non-production change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		got, err := cfg.NextVersion("v1.2.3", 1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, "v1.2.4", got)
	})

	t.Run("minor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "minor change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		got, err := cfg.NextVersion("v1.2.3", 1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, "v1.3.0", got)
	})

	t.Run("major", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "breaking change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		got, err := cfg.NextVersion("v1.2.3", 1, 2, 3)
		require.NoError(t, err)
		require.Equal(t, "v2.0.0", got)
	})

	t.Run("invalid current version", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar", "breaking change"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		_, err := cfg.NextVersion("limabeans", 1, 2, 3)
		require.Error(t, err)
	})

	t.Run("missing label", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)
		mockFetcher := mocks.NewMockPRLabelFetcher(ctrl)
		mockFetcher.EXPECT().FetchPRLabels(1).Return([]string{"foo", "bar"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(2).Return([]string{"Baz", "QUX", "Patch"}, nil)
		mockFetcher.EXPECT().FetchPRLabels(3).Return([]string{"non-production change"}, nil)
		cfg := &Config{
			PRLabelFetcher: mockFetcher,
			RequireLabels:  true,
		}
		wantErr := &PRMissingLabelErr{
			IDs: []int{1},
		}
		_, err := cfg.NextVersion("v1.2.3", 1, 2, 3)
		require.Error(t, err)
		require.Equal(t, wantErr, err)
	})
}

func Test_nextVersion(t *testing.T) {
	mustNextVersion := func(prev string, bump VersionChange) string {
		t.Helper()
		got, err := nextVersion(prev, bump)
		require.NoError(t, err)
		return got
	}
	require.Equal(t, "v1.2.3", mustNextVersion("v1.2.2", VersionChangePatch))
	require.Equal(t, "v1.2.2", mustNextVersion("v1.2.2", VersionChangeNone))
	require.Equal(t, "v1.3.0", mustNextVersion("v1.2.2", VersionChangeMinor))
	require.Equal(t, "v2.0.0", mustNextVersion("v1.2.2", VersionChangeMajor))
	require.Equal(t, "v2.0.0", mustNextVersion("v1", VersionChangeMajor))
	require.Equal(t, "v1.1.0", mustNextVersion("v1", VersionChangeMinor))
	require.Equal(t, "v0.1.0", mustNextVersion("v0", VersionChangeMinor))
}
