package model

import (
	"encoding/json"
	"testing"
)

func TestMalwareNameUsesSignatureBeforeClassification(t *testing.T) {
	var alert Alert
	err := json.Unmarshal([]byte(`{"file":{"Ext":{"malware_classification":"Malicious","malware_signature":["Example.Test.Malware"]}}}`), &alert)
	if err != nil {
		t.Fatal(err)
	}
	if got := alert.MalwareName(); got != "Example.Test.Malware" {
		t.Fatalf("malware name = %q", got)
	}
	alert.File.Ext.MalwareSignature = json.RawMessage(`[]`)
	if got := alert.MalwareName(); got != "Malicious" {
		t.Fatalf("classification = %q", got)
	}
}

func TestMalwareSignatureObjectAndMLScoreAreAccepted(t *testing.T) {
	var alert Alert
	err := json.Unmarshal([]byte(`{"file":{"Ext":{"malware_classification":{"score":0.99,"version":"1"},"malware_signature":{"primary":{"threat_name":"Windows.Test.Threat"}}}}}`), &alert)
	if err != nil {
		t.Fatal(err)
	}
	if got := alert.MalwareName(); got != "Windows.Test.Threat" {
		t.Fatalf("signature = %q", got)
	}
	alert.File.Ext.MalwareSignature = nil
	if got := alert.MalwareName(); got != "" {
		t.Fatalf("score mistaken for name: %q", got)
	}
}
