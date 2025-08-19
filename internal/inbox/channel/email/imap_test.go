package email

import (
	"bytes"
	"os"
	"testing"

	"github.com/jhillyerd/enmime"
)

func TestExtractMessageIDFromHeaders(t *testing.T) {
	tests := []struct {
		name           string
		emlFile        string
		expectedResult string
		description    string
	}{
		{
			name:           "Normal Message ID",
			emlFile:        "testdata/normal_message_id.eml",
			expectedResult: "normal123@example.com",
			description:    "Standard Message-ID format should be extracted correctly",
		},
		{
			name:           "Double @ Message ID",
			emlFile:        "testdata/double_at_message_id.eml",
			expectedResult: "message123@username@example.org",
			description:    "Message-ID with multiple @ symbols should be extracted correctly",
		},
		{
			name:           "Triple @ Message ID",
			emlFile:        "testdata/triple_at_message_id.eml",
			expectedResult: "id@user@company@domain.com",
			description:    "Message-ID with three @ symbols should be extracted correctly",
		},
		{
			name:           "Missing Message ID",
			emlFile:        "testdata/missing_message_id.eml",
			expectedResult: "",
			description:    "Email without Message-ID header should return empty string",
		},
		{
			name:           "Whitespace Message ID",
			emlFile:        "testdata/whitespace_message_id.eml",
			expectedResult: "whitespace123@username@example.org",
			description:    "Message-ID with whitespace and multiple @ symbols should be cleaned and extracted",
		},
		{
			name:           "No Brackets Message ID",
			emlFile:        "testdata/no_brackets_message_id.eml",
			expectedResult: "nobrackets123@username@example.org",
			description:    "Message-ID without angle brackets should be extracted correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the email file
			emlData, err := os.ReadFile(tt.emlFile)
			if err != nil {
				t.Fatalf("Failed to read email file %s: %v", tt.emlFile, err)
			}

			// Parse the email with enmime
			envelope, err := enmime.ReadEnvelope(bytes.NewReader(emlData))
			if err != nil {
				t.Fatalf("Failed to parse email: %v", err)
			}

			// Test the actual function
			result := extractMessageIDFromHeaders(envelope)

			// Verify the result
			if result != tt.expectedResult {
				t.Errorf("extractMessageIDFromHeaders() = %q, want %q\nDescription: %s",
					result, tt.expectedResult, tt.description)
			}

			// Log the raw header for debugging
			rawHeader := envelope.GetHeader(headerMessageID)
			t.Logf("Raw Message-ID header: %q", rawHeader)
			t.Logf("Extracted Message-ID: %q", result)
		})
	}
}

func TestProblematicMessageIDScenarios(t *testing.T) {
	// Test Message-ID with multiple @ symbols that cause go-imap parsing to fail
	t.Run("Multiple @ Symbols in Message-ID", func(t *testing.T) {
		emlData, err := os.ReadFile("testdata/double_at_message_id.eml")
		if err != nil {
			t.Fatalf("Failed to read email file: %v", err)
		}

		envelope, err := enmime.ReadEnvelope(bytes.NewReader(emlData))
		if err != nil {
			t.Fatalf("Failed to parse email: %v", err)
		}

		result := extractMessageIDFromHeaders(envelope)
		expected := "message123@username@example.org"

		if result != expected {
			t.Errorf("Failed to extract Message-ID with multiple @ symbols. Got %q, want %q", result, expected)
		}

		// Verify it's not empty (which would cause conversation corruption)
		if result == "" {
			t.Error("Extracted Message-ID is empty, this would cause conversation corruption")
		}
	})

	// Test various Message-ID formatting edge cases
	t.Run("Message-ID Formatting Edge Cases", func(t *testing.T) {
		testCases := []struct {
			file        string
			description string
		}{
			{"testdata/triple_at_message_id.eml", "Message-ID with three @ symbols"},
			{"testdata/whitespace_message_id.eml", "Message-ID with extra whitespace"},
			{"testdata/no_brackets_message_id.eml", "Message-ID without angle brackets"},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				emlData, err := os.ReadFile(tc.file)
				if err != nil {
					t.Fatalf("Failed to read email file %s: %v", tc.file, err)
				}

				envelope, err := enmime.ReadEnvelope(bytes.NewReader(emlData))
				if err != nil {
					t.Fatalf("Failed to parse email %s: %v", tc.file, err)
				}

				result := extractMessageIDFromHeaders(envelope)

				// Ensure Message-ID extraction succeeds
				if result == "" {
					t.Errorf("Message-ID extraction failed for %s, got empty string", tc.file)
				}

				// Ensure proper cleaning - no angle brackets or whitespace
				if result != "" {
					if result[0] == '<' || result[len(result)-1] == '>' {
						t.Errorf("Message-ID still contains angle brackets: %q", result)
					}
					if result[0] == ' ' || result[len(result)-1] == ' ' {
						t.Errorf("Message-ID still contains whitespace: %q", result)
					}
				}
			})
		}
	})
}
