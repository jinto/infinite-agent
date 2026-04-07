package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tier 1: 무료, 매 커밋 — SKILL.md 정적 검증
// 필수 섹션이 존재하는지, 참조가 일관되는지 확인

var requiredSections = []string{
	"## 언제 사용",
	"## 사용하지 말 것",
	"## 입출력",
}

var expectedSkills = []string{
	"autopilot", "think", "plan", "build", "review",
	"research", "design", "guard", "test", "ship",
}

func TestAllSkillsExist(t *testing.T) {
	for _, skill := range expectedSkills {
		path := filepath.Join("..", "skills", skill, "SKILL.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("skill %q missing: %s", skill, path)
		}
	}
}

func TestRemovedSkillsGone(t *testing.T) {
	removed := []string{"pitch", "sync"}
	for _, skill := range removed {
		path := filepath.Join("..", "skills", skill)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("removed skill %q still exists: %s", skill, path)
		}
	}
}

func TestSkillRequiredSections(t *testing.T) {
	for _, skill := range expectedSkills {
		if skill == "guard" {
			continue // guard is a rule set, not user-invocable
		}
		path := filepath.Join("..", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("cannot read %s: %v", path, err)
			continue
		}
		content := string(data)
		for _, section := range requiredSections {
			if !strings.Contains(content, section) {
				t.Errorf("skill %q missing section %q", skill, section)
			}
		}
	}
}

func TestSkillFrontmatter(t *testing.T) {
	for _, skill := range expectedSkills {
		path := filepath.Join("..", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("cannot read %s: %v", path, err)
			continue
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("skill %q missing frontmatter", skill)
			continue
		}
		if !strings.Contains(content, "name: "+skill) {
			t.Errorf("skill %q frontmatter name mismatch", skill)
		}
		if !strings.Contains(content, "description:") {
			t.Errorf("skill %q missing description in frontmatter", skill)
		}
	}
}

func TestNoStaleReferences(t *testing.T) {
	staleRefs := []string{"/ina:pitch", "/ina:sync", "eidaa"}
	for _, skill := range expectedSkills {
		path := filepath.Join("..", "skills", skill, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		for _, ref := range staleRefs {
			if strings.Contains(content, ref) {
				t.Errorf("skill %q contains stale reference %q", skill, ref)
			}
		}
	}
}

func TestEvalFixturesExist(t *testing.T) {
	data, err := os.ReadFile("eval_scenarios.json")
	if err != nil {
		t.Skip("eval_scenarios.json not found — eval not configured")
	}
	var scenarios []struct {
		Name    string `json:"name"`
		Fixture string `json:"fixture"`
	}
	if err := json.Unmarshal(data, &scenarios); err != nil {
		t.Fatalf("parse eval_scenarios.json: %v", err)
	}
	for _, sc := range scenarios {
		path := filepath.Join("..", sc.Fixture)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("eval scenario %q: fixture missing: %s", sc.Name, sc.Fixture)
		}
	}
}

func TestCLAUDEmdConsistency(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("cannot read CLAUDE.md: %v", err)
	}
	content := string(data)
	for _, skill := range expectedSkills {
		ref := "/ina:" + skill
		if !strings.Contains(content, ref) && skill != "guard" {
			t.Errorf("CLAUDE.md missing reference to %s", ref)
		}
	}
}
