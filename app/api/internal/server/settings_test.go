package server

import "testing"

func TestValidateSetting_Int(t *testing.T) {
	spec := settingSpec{Key: "retention_days", Type: "int"}

	// float64 que é inteiro (JSON numbers vêm como float64)
	val, err := validateSetting(spec, float64(30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "30" {
		t.Errorf("got %q, want '30'", val)
	}

	// float64 com decimal deve falhar
	_, err = validateSetting(spec, float64(30.5))
	if err == nil {
		t.Error("expected error for non-integer float64")
	}

	// string deve falhar
	_, err = validateSetting(spec, "30")
	if err == nil {
		t.Error("expected error for string type")
	}
}

func TestValidateSetting_Float(t *testing.T) {
	spec := settingSpec{Key: "outlier_threshold", Type: "float"}

	val, err := validateSetting(spec, float64(3.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "3.5" {
		t.Errorf("got %q, want '3.5'", val)
	}

	// int deve ser aceito como float
	val, err = validateSetting(spec, 3)
	if err != nil {
		t.Fatalf("unexpected error for int: %v", err)
	}
	if val != "3" {
		t.Errorf("got %q, want '3'", val)
	}
}

func TestValidateSetting_UnknownKey(t *testing.T) {
	_, ok := findSettingSpec("nonexistent_key")
	if ok {
		t.Error("expected findSettingSpec to return false for unknown key")
	}

	// Chaves conhecidas
	knownKeys := []string{
		"collect_interval", "retention_days", "outlier_threshold",
		"leak_r2_threshold", "leak_daily_mb_threshold", "analysis_window_days",
		"cluster_interval", "storage_interval", "stale_service_days",
	}
	for _, key := range knownKeys {
		if _, ok := findSettingSpec(key); !ok {
			t.Errorf("findSettingSpec(%q) returned false, expected true", key)
		}
	}
}

func TestValidateSetting_IntRejectsFloatWithDecimal(t *testing.T) {
	spec := settingSpec{Key: "test", Type: "int"}

	// 30.0 deve ser aceito (é inteiro em float64)
	_, err := validateSetting(spec, float64(30.0))
	if err != nil {
		t.Errorf("30.0 should be accepted as int, got error: %v", err)
	}

	// 30.1 deve ser rejeitado
	_, err = validateSetting(spec, float64(30.1))
	if err == nil {
		t.Error("30.1 should be rejected as int")
	}
}
