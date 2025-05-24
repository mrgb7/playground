package logger

import (
	"testing"

	"github.com/fatih/color"
)

func TestLoggerFunctions(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Test that functions don't panic when called
	t.Run("Info", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info function panicked: %v", r)
			}
		}()
		Info("test message %s", "arg")
	})

	t.Run("Infoln", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Infoln function panicked: %v", r)
			}
		}()
		Infoln("test message %s", "arg")
	})

	t.Run("Warn", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn function panicked: %v", r)
			}
		}()
		Warn("warning: %s", "test")
	})

	t.Run("Warnln", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warnln function panicked: %v", r)
			}
		}()
		Warnln("warning: %s", "test")
	})

	t.Run("Error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error function panicked: %v", r)
			}
		}()
		Error("error: %s", "test")
	})

	t.Run("Errorln", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Errorln function panicked: %v", r)
			}
		}()
		Errorln("error: %s", "test")
	})

	t.Run("Debug", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug function panicked: %v", r)
			}
		}()
		Debug("debug: %s", "test")
	})

	t.Run("Debugln", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debugln function panicked: %v", r)
			}
		}()
		Debugln("debug: %s", "test")
	})

	t.Run("Success", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Success function panicked: %v", r)
			}
		}()
		Success("success: %s", "test")
	})

	t.Run("Successln", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Successln function panicked: %v", r)
			}
		}()
		Successln("success: %s", "test")
	})

	t.Run("Print", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Print function panicked: %v", r)
			}
		}()
		Print("plain: %s", "test")
	})

	t.Run("Println", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Println function panicked: %v", r)
			}
		}()
		Println("plain: %s", "test")
	})
}
