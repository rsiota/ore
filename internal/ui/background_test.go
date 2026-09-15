package ui

import (
	"strings"
	"testing"
)

func TestPaintBgRespectsTransparentSetting(t *testing.T) {
	defer applyTheme(defaultThemeName)
	applyTheme("light")

	in := "hello"
	opaque := Model{transparentBg: false}.paintBg(in)
	if !strings.HasPrefix(opaque, "\x1b[") {
		t.Fatalf("opaque should paint theme bg, got %q", opaque)
	}
	if strings.HasPrefix(opaque, defaultBgResetSeq) {
		t.Fatalf("opaque should not emit bg reset: %q", opaque)
	}

	transparent := Model{transparentBg: true}.paintBg(in)
	if transparent != defaultBgResetSeq+in {
		t.Fatalf("transparent should reset default bg, got %q", transparent)
	}
}

func TestExSetTransparentBackground(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := Model{config: nil}
	_ = m.exSet(nil)
	if !strings.Contains(m.status, "transparent_background=off") {
		t.Fatalf("status = %q", m.status)
	}

	_ = m.exSet([]string{"transparent_background", "on"})
	if !m.transparentBg || m.config == nil || !m.config.TransparentBackground {
		t.Fatalf("not enabled: transparentBg=%v config=%#v", m.transparentBg, m.config)
	}
	if m.status != "transparent_background=on" {
		t.Fatalf("status = %q", m.status)
	}

	_ = m.exSet([]string{"transparent", "off"})
	if m.transparentBg || m.config.TransparentBackground {
		t.Fatal("expected off")
	}
}
