package engineering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlan_Layers(t *testing.T) {
	p, err := BuildPlan(Brief{
		Style:       StyleLayers,
		Feature:     "Pricing",
		Description: "Calcula preço final",
		PackagePath: "internal/service",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Interfaces) != 1 {
		t.Fatalf("interfaces: %d", len(p.Interfaces))
	}
	if p.Interfaces[0].Name == "" {
		t.Fatal("nome vazio")
	}
	files, err := GenerateFiles(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 3 {
		t.Fatalf("files: %d", len(files))
	}
	for path, content := range files {
		if !strings.Contains(content, "package ") {
			t.Fatalf("%s sem package", path)
		}
	}
}

func TestWriteScaffold(t *testing.T) {
	dir := t.TempDir()
	p, err := BuildPlan(Brief{
		Style:       StyleLayers,
		Feature:     "Echo",
		Description: "eco",
		PackagePath: "internal/service",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := WriteScaffold(dir, p, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) == 0 {
		t.Fatal("nada escrito")
	}
	for _, w := range res.Written {
		if _, err := os.Stat(filepath.Join(dir, w)); err != nil {
			t.Fatal(err)
		}
	}
}
