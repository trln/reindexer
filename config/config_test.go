package config

import (
    "testing"
)


func TestLoadConfigBadParamWorkerValue(t *testing.T) {
    path := "testdata/config_negative_workers.json"
    _, err := LoadConfig(path)
    if err == nil {
        t.Errorf("should not have been able to load a configuration with negative workers")
    }
}





