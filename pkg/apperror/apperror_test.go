package apperror

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "account not found")
	if err == nil {
		t.Fatal("New must not return nil")
	}
	if err.Code != CodeNotFound || err.Msg != "account not found" || err.Err != nil {
		t.Errorf("unexpected AppError: %+v", err)
	}
	if err.Error() != "account not found" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestWrap_nilErr(t *testing.T) {
	err := Wrap(CodeInternal, "db failed", nil)
	if err.Err != nil {
		t.Errorf("Wrap(nil) should store nil: %v", err.Err)
	}
}

func TestWrap_withErr(t *testing.T) {
	root := errors.New("connection refused")
	err := Wrap(CodeInternal, "db failed", root)
	if err.Err != root {
		t.Errorf("Wrap should preserve cause")
	}
	if !errors.Is(err, root) {
		t.Error("errors.Is(err, root) should be true")
	}
}

func TestCodeOf(t *testing.T) {
	if c := CodeOf(nil); c != CodeInternal {
		t.Errorf("CodeOf(nil) = %v, want CodeInternal", c)
	}
	if c := CodeOf(errors.New("random")); c != CodeInternal {
		t.Errorf("CodeOf(random) = %v, want CodeInternal", c)
	}
	if c := CodeOf(New(CodeNotFound, "x")); c != CodeNotFound {
		t.Errorf("CodeOf(NotFound) = %v", c)
	}
	wrapped := Wrap(CodeInvalidArgument, "bad", errors.New("x"))
	if c := CodeOf(wrapped); c != CodeInvalidArgument {
		t.Errorf("CodeOf(wrapped) = %v", c)
	}
}

func TestUnwrap(t *testing.T) {
	root := errors.New("root")
	err := Wrap(CodeInternal, "wrap", root)
	if errors.Unwrap(err) != root {
		t.Error("Unwrap should return root")
	}
}
