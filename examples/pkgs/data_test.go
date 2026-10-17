package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestRegistryIsConsistent(t *testing.T) {
	ps := registry()
	seen := map[string]bool{}
	for _, p := range ps {
		if seen[p.name] {
			t.Errorf("%s appears twice", p.name)
		}
		seen[p.name] = true
		if p.latest == "" || p.sizeKB <= 0 || p.desc == "" {
			t.Errorf("%s is incomplete: %+v", p.name, p)
		}
		for _, d := range p.deps {
			if index(ps, d) < 0 {
				t.Errorf("%s needs %s, which is not in the registry", p.name, d)
			}
		}
		if p.installed != "" {
			for _, d := range p.deps {
				if ps[index(ps, d)].installed == "" {
					t.Errorf("%s is installed but needs %s, which is not", p.name, d)
				}
			}
		}
	}
	if len(ps) < 25 {
		t.Errorf("only %d packages; the pager needs more than a page", len(ps))
	}
}

func TestStatusAndSize(t *testing.T) {
	if (pkg{latest: "1"}).status() != available || (pkg{latest: "1", installed: "1"}).status() != installedOK || (pkg{latest: "2", installed: "1"}).status() != updatable {
		t.Error("status wrong")
	}
	if size(24) != "24 KB" || size(1480) != "1.4 MB" || size(1024) != "1.0 MB" {
		t.Errorf("size = %s %s %s", size(24), size(1480), size(1024))
	}
	if status(1).String() != "installed" {
		t.Error("status text")
	}
}

func TestInstallOrderPutsDependenciesFirst(t *testing.T) {
	ps := registry()
	order := installOrder(ps, index(ps, "configure"), nil)
	got := joinNames(ps, order)
	// configure needs tomlish and textwrap; textwrap is installed.
	if got != "tomlish, configure" {
		t.Fatalf("order = %s", got)
	}
	order = installOrder(ps, index(ps, "gitlite"), nil)
	if joinNames(ps, order) != "zlibx, gitlite" {
		t.Fatalf("order = %s", joinNames(ps, order))
	}
	if got := installOrder(ps, index(ps, "argon"), nil); len(got) != 0 {
		t.Fatalf("an installed package needs %v", got)
	}
	skip := map[int]bool{index(ps, "tomlish"): true}
	if got := joinNames(ps, installOrder(ps, index(ps, "configure"), skip)); got != "configure" {
		t.Fatalf("with tomlish queued: %s", got)
	}
}

func TestInstallOrderSurvivesACycle(t *testing.T) {
	ps := []pkg{{name: "a", deps: []string{"b"}}, {name: "b", deps: []string{"a"}}}
	if got := installOrder(ps, 0, nil); len(got) != 2 {
		t.Fatalf("order = %v", got)
	}
}

func TestDependentsListsOnlyInstalledPackages(t *testing.T) {
	ps := registry()
	if got := dependents(ps, index(ps, "textwrap")); !reflect.DeepEqual(got, []string{"argon"}) {
		t.Fatalf("dependents of textwrap = %v", got)
	}
	if got := dependents(ps, index(ps, "colorize")); len(got) != 0 {
		t.Fatalf("logr is not installed but colorize has dependents %v", got)
	}
	if !strings.Contains(joinNames(ps, []int{0, 1}), "argon, bytesize") {
		t.Fatal("joinNames")
	}
}
