package download

import "testing"

func Test_decompose(t *testing.T) {
	var domain, repo, tag string
	name := "busybox"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "" || repo != "busybox" || tag != "" {
		t.Fail()
	}

	name = "busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "" || repo != "busybox" || tag != "latest" {
		t.Fail()
	}
	name = "library/busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "" || repo != "library/busybox" || tag != "latest" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}
	name = "localhost:5000/busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "localhost:5000" || repo != "busybox" || tag != "latest" {
		t.Fatal(domain, repo, tag)
	}

	name = "localhost/busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "localhost" || repo != "busybox" || tag != "latest" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}

	name = "localhost/busybox"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "localhost" || repo != "busybox" || tag != "" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}

	name = "dockerhub.io/busybox"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "dockerhub.io" || repo != "busybox" || tag != "" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}

	name = "dockerhub.io/busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "dockerhub.io" || repo != "busybox" || tag != "latest" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}
	name = "dockerhub.io/lib/busybox:latest"
	domain, repo, tag = decomposeRepoTag(name)
	if domain != "dockerhub.io" || repo != "lib/busybox" || tag != "latest" {
		t.Fatal("domain:", domain, "repo:", repo, "tag:", tag)
	}
}
