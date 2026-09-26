package aiscript

import "testing"

func TestStatusForEmpty(t *testing.T) {
	st := StatusFor("")
	if st.Available {
		t.Fatal("empty model must not be available")
	}
	if st.Stage != "S0" {
		t.Fatalf("stage=%q", st.Stage)
	}
	if st.Reason == "" {
		t.Fatal("expected English reason")
	}
}

func TestStatusForPathStillS0(t *testing.T) {
	st := StatusFor("/tmp/fake.onnx")
	if st.Available {
		t.Fatal("S0 must refuse even when a path is set")
	}
	if st.ModelPath != "/tmp/fake.onnx" {
		t.Fatalf("modelPath=%q", st.ModelPath)
	}
}

func TestDraftFailsClosed(t *testing.T) {
	_, err := Draft(DraftRequest{VideoPath: "/tmp/x.mp4"})
	if err == nil {
		t.Fatal("expected error")
	}
}
