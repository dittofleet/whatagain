package terrier

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestCollectKeepsOnlyReposWhatagainCanName(t *testing.T) {
	// The shape terrier documents for `ls --json`, with the two kinds of
	// entry that are no use here: one whose directory has gone, and one
	// with no GitHub origin to name it by.
	const registry = `{"projects":[
		{"path":"/s/dittofleet/whatagain","slug":"dittofleet/whatagain"},
		{"path":"/s/cli/terrier","slug":"sylophi/terrier"},
		{"path":"/s/scratch"},
		{"path":"/s/gone","missing":true}
	]}`

	var got struct {
		Projects []project `json:"projects"`
	}
	if err := json.Unmarshal([]byte(registry), &got); err != nil {
		t.Fatalf("unmarshal errored: %v", err)
	}

	want := []string{"dittofleet/whatagain", "sylophi/terrier"}
	if slugs := collect(got.Projects); !slices.Equal(slugs, want) {
		t.Errorf("collect = %v, want %v", slugs, want)
	}
}

func TestCollectDropsTheSameRepoTwice(t *testing.T) {
	// Terrier registers paths, so a repo cloned twice is two entries
	// there. It is one project here, and the casing GitHub does not care
	// about must not make it two either.
	projects := []project{
		{Slug: "dittofleet/whatagain"},
		{Slug: "dittofleet/whatagain"},
		{Slug: "Dittofleet/Whatagain"},
	}
	if got := collect(projects); !slices.Equal(got, []string{"dittofleet/whatagain"}) {
		t.Errorf("collect = %v, want the repo once", got)
	}
}
