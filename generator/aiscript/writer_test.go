package aiscript

import "testing"

func TestStatusForEmpty(t *testing.T) {
	st := StatusFor("")
	if st.Available {
		t.Fatal("empty model must not be available")
	}
	if st.Stage != "S1" {
		t.Fatalf("stage=%q", st.Stage)
	}
	if st.Reason == "" {
		t.Fatal("expected English reason")
	}
}

func TestStatusForPathStillClosed(t *testing.T) {
	st := StatusFor("/tmp/fake.onnx")
	if st.Available {
		t.Fatal("must refuse even when a path is set until S2")
	}
	if st.ModelPath != "/tmp/fake.onnx" {
		t.Fatalf("modelPath=%q", st.ModelPath)
	}
	if st.Stage != "S1" {
		t.Fatalf("stage=%q", st.Stage)
	}
}

func TestDraftFailsClosed(t *testing.T) {
	_, err := Draft(DraftRequest{VideoPath: "/tmp/x.mp4"})
	if err == nil {
		t.Fatal("expected error")
	}
}
