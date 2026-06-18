package documents

import "testing"

func TestSafeFilename(t *testing.T) {
	tests := map[string]string{
		"blood results.pdf": "blood_results.pdf",
		"../../scan.png":    "scan.png",
		"  ???  ":           "document",
	}
	for input, want := range tests {
		if got := safeFilename(input); got != want {
			t.Fatalf("safeFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewStorageName(t *testing.T) {
	name, err := newStorageName("Lab Result.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(name) <= len("-Lab_Result.pdf") {
		t.Fatalf("storage name too short: %q", name)
	}
	if name[len(name)-len("Lab_Result.pdf"):] != "Lab_Result.pdf" {
		t.Fatalf("storage name did not preserve safe filename: %q", name)
	}
}
