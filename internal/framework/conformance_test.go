package framework

import "testing"

func TestFrameworkProfilesConformance(t *testing.T) {
	subtrees := []string{
		"src/app/(marketing)/blog/[slug]",
		"src/routes/:id",
		"src/routes/$teamId",
		"src/features/account",
	}
	files := []string{
		"src/app/(marketing)/blog/[slug]/page.tsx",
		"src/routes/teams.$teamId.tsx",
		"src/screens/Home.ios.tsx",
		"src/app/users-routing.module.ts",
	}

	profiles := RegisteredProfiles()
	if len(profiles) == 0 {
		t.Fatal("expected at least one registered profile")
	}

	for _, profile := range profiles {
		if profile.ID() == "" {
			t.Fatalf("profile must have non-empty id: %+v", profile)
		}
		for _, subtree := range subtrees {
			once := profile.NormalizeSubtree(subtree)
			twice := profile.NormalizeSubtree(once)
			if once != twice {
				t.Fatalf("profile %s subtree normalization not idempotent: once=%q twice=%q", profile.ID(), once, twice)
			}
		}
		for _, file := range files {
			once := profile.NormalizeFile(file)
			twice := profile.NormalizeFile(once)
			if once != twice {
				t.Fatalf("profile %s file normalization not idempotent: once=%q twice=%q", profile.ID(), once, twice)
			}
		}
	}
}
