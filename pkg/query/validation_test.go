package query

import "testing"

func TestValidateReadOnlyQueryAcceptsSelectAfterComments(t *testing.T) {
	err := ValidateReadOnlyQuery("-- note\n/* block */\nSELECT 1;")
	if err != nil {
		t.Fatalf("ValidateReadOnlyQuery returned error: %v", err)
	}
}

func TestValidateReadOnlyQueryRejectsWriteStatements(t *testing.T) {
	err := ValidateReadOnlyQuery("INSERT INTO t VALUES (1)")
	if err == nil {
		t.Fatalf("ValidateReadOnlyQuery returned nil, want error")
	}
}

func TestNormalizeQueryForValidationRejectsMultipleStatements(t *testing.T) {
	_, err := NormalizeQueryForValidation("SELECT 1; SELECT 2;")
	if err == nil {
		t.Fatalf("NormalizeQueryForValidation returned nil, want error")
	}
}
