package global

import (
	"testing"
)

func TestFilterRepositories(t *testing.T) {
	repos := []RepoInfo{
		{Path: "/root/github.com/user/bar", RelativePath: "github.com/user/bar", Name: "bar"},
		{Path: "/root/github.com/user/foobar", RelativePath: "github.com/user/foobar", Name: "foobar"},
		{Path: "/root/github.com/user/bazbar", RelativePath: "github.com/user/bazbar", Name: "bazbar"},
		{Path: "/root/github.com/org/unrelated", RelativePath: "github.com/org/unrelated", Name: "unrelated"},
	}

	t.Run("empty query returns all repos unchanged", func(t *testing.T) {
		result := FilterRepositories(repos, "")
		if len(result) != len(repos) {
			t.Errorf("expected %d repos, got %d", len(repos), len(result))
		}
	})

	t.Run("filters by substring match", func(t *testing.T) {
		result := FilterRepositories(repos, "bar")

		// Should contain bar, foobar, bazbar but not unrelated
		if len(result) != 3 {
			t.Errorf("expected 3 repos, got %d", len(result))
		}

		names := make(map[string]bool)
		for _, r := range result {
			names[r.Name] = true
		}

		if !names["bar"] {
			t.Error("expected 'bar' in results")
		}
		if !names["foobar"] {
			t.Error("expected 'foobar' in results")
		}
		if !names["bazbar"] {
			t.Error("expected 'bazbar' in results")
		}
		if names["unrelated"] {
			t.Error("'unrelated' should not be in results")
		}
	})

	t.Run("prefix matches come before substring matches", func(t *testing.T) {
		result := FilterRepositories(repos, "bar")

		// "bar" is a prefix match (name starts with "bar")
		// "foobar" and "bazbar" are substring matches (name contains "bar" but doesn't start with it)
		if len(result) < 1 {
			t.Fatal("expected at least 1 result")
		}

		// First result should be the prefix match "bar"
		if result[0].Name != "bar" {
			t.Errorf("expected first result to be 'bar' (prefix match), got '%s'", result[0].Name)
		}

		// Remaining results should be substring matches
		for i := 1; i < len(result); i++ {
			if result[i].Name == "bar" {
				t.Errorf("prefix match 'bar' should not appear after index 0, found at index %d", i)
			}
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		result := FilterRepositories(repos, "BAR")

		if len(result) != 3 {
			t.Errorf("expected 3 repos with case-insensitive match, got %d", len(result))
		}
	})

	t.Run("matches against relative path", func(t *testing.T) {
		result := FilterRepositories(repos, "github.com/user")

		// All repos under github.com/user should match
		if len(result) != 3 {
			t.Errorf("expected 3 repos matching path, got %d", len(result))
		}
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		result := FilterRepositories(repos, "nonexistent")

		if len(result) != 0 {
			t.Errorf("expected 0 repos, got %d", len(result))
		}
	})
}

func TestFilterRepositories_PrefixPriority(t *testing.T) {
	// Test case specifically for prefix priority with multiple prefix matches
	repos := []RepoInfo{
		{Path: "/root/test", RelativePath: "test", Name: "test"},
		{Path: "/root/testing", RelativePath: "testing", Name: "testing"},
		{Path: "/root/feat/test", RelativePath: "feat/test", Name: "test"},
		{Path: "/root/mytest", RelativePath: "mytest", Name: "mytest"},
	}

	t.Run("multiple prefix matches all come before substring matches", func(t *testing.T) {
		result := FilterRepositories(repos, "test")

		// "test" and "testing" are prefix matches (name starts with "test")
		// "feat/test" has name "test" which is a prefix match
		// "mytest" is a substring match (name contains "test" but doesn't start with it)

		if len(result) != 4 {
			t.Errorf("expected 4 results, got %d", len(result))
		}

		// Find where "mytest" appears (should be last as it's a substring match)
		mytestIdx := -1
		for i, r := range result {
			if r.Name == "mytest" {
				mytestIdx = i
				break
			}
		}

		if mytestIdx == -1 {
			t.Fatal("'mytest' not found in results")
		}

		// All items before mytest should be prefix matches
		for i := 0; i < mytestIdx; i++ {
			name := result[i].Name
			if name != "test" && name != "testing" {
				t.Errorf("expected prefix match at index %d, got '%s'", i, name)
			}
		}
	})
}
