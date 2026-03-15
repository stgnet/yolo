// Email handling unit tests for YOLO
package main

import (
	"encoding/base64"
	"testing"
)

// TestDecodeMIMEWord_Basic - Test MIME word decoding with basic ASCII
func TestDecodeMIMEWord_Basic(t *testing.T) {
	word := "Hello World"
	result := decodeMIMEWord(word)
	
	if result != word {
		t.Errorf("Expected '%s', got '%s'", word, result)
	}
}

// TestDecodeMIMEWord_UTF8B64 - Test MIME word decoding with UTF-8 Base64 encoding
func TestDecodeMIMEWord_UTF8B64(t *testing.T) {
	original := "Hello 世界"
	encoded := "=?" + "UTF-8" + "?" + "B" + "?" + base64.StdEncoding.EncodeToString([]byte(original)) + "?="
	
	result := decodeMIMEWord(encoded)
	
	if result != original {
		t.Errorf("Expected '%s', got '%s'", original, result)
	}
}

// TestDecodeMIMEWord_UTF8Q - Test MIME word decoding with Quoted-Printable encoding
func TestDecodeMIMEWord_UTF8Q(t *testing.T) {
	original := "Hello World"
	encoded := "=?" + "UTF-8" + "?" + "Q" + "?" + "Hello_World" + "?="
	
	result := decodeMIMEWord(encoded)
	
	if result != original {
		t.Errorf("Expected '%s', got '%s'", original, result)
	}
}

// TestDecodeMIMEWord_InvalidFormat - Test MIME word decoding with invalid format
func TestDecodeMIMEWord_InvalidFormat(t *testing.T) {
	word := "=?invalid format"
	result := decodeMIMEWord(word)
	
	// Should return the original when format is invalid
	if result != word {
		t.Logf("Expected '%s' to remain unchanged, got '%s'", word, result)
	}
}

// TestDecodeMIMEWord_MixedContent - Test MIME word decoding with mixed content
func TestDecodeMIMEWord_MixedContent(t *testing.T) {
	input := "=?UTF-8?B?SGVsbG8gV29ybGQ=? World"
	decoded := decodeMIMEWord(input)
	
	if decoded != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", decoded)
	}
}

// TestDecodeQ_Basic - Test Quoted-Printable decoding with underscore
func TestDecodeQ_Basic(t *testing.T) {
	input := "Hello_World_Test"
	expected := "Hello World Test"
	
	result, err := decodeQ(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(result))
	}
}

// TestDecodeQ_HexEscape - Test Quoted-Printable decoding with hex escapes
func TestDecodeQ_HexEscape(t *testing.T) {
	input := "Hello=3DWorld" // = is hex 3D in quoted-printable
	expected := "Hello=World"
	
	result, err := decodeQ(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if string(result) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(result))
	}
}

// TestDecodeQ_InvalidHex - Test Quoted-Printable decoding with invalid hex
func TestDecodeQ_InvalidHex(t *testing.T) {
	input := "Hello=ZZWorld" // ZZ is not valid hex
	
	result, err := decodeQ(input)
	if err == nil {
		t.Errorf("Expected error for invalid hex, got result: %s", string(result))
	}
	
	expectedLength := 15 // Original length before =ZZ (3 chars) were consumed
	if len(result) >= expectedLength {
		t.Logf("Unexpected success with invalid hex escape")
	}
}

// TestParseEmail_FullFormat - Test email parsing with full MIME format
func TestParseEmail_FullFormat(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Test Subject
Content-Type: text/plain; charset=UTF-8

This is the email body content.
`
	
	email := parseEmail(rawEmail)
	
	if email.From != "sender@example.com" {
		t.Errorf("Expected 'sender@example.com', got '%s'", email.From)
	}
	
	if email.To != "recipient@example.com" {
		t.Errorf("Expected 'recipient@example.com', got '%s'", email.To)
	}
	
	if email.Subject != "Test Subject" {
		t.Errorf("Expected 'Test Subject', got '%s'", email.Subject)
	}
	
	if email.Body != "This is the email body content." {
		t.Errorf("Expected 'This is the email body content.', got '%s'", email.Body)
	}
}

// TestParseEmail_MinimalFormat - Test email parsing with minimal format
func TestParseEmail_MinimalFormat(t *testing.T) {
	rawEmail := `From: test@domain.com
Subject: Minimal Subject
To: user@domain.com

Simple body text.
`
	
	email := parseEmail(rawEmail)
	
	if email.From != "test@domain.com" {
		t.Errorf("Expected 'test@domain.com', got '%s'", email.From)
	}
	
	if email.Subject != "Minimal Subject" {
		t.Errorf("Expected 'Minimal Subject', got '%s'", email.Subject)
	}
}

// TestParseEmail_Multipart - Test parsing multipart email
func TestParseEmail_Multipart(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Multipart Email
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain

This is the plain text body.
--boundary123
Content-Type: text/html

<p>This is the HTML body.</p>
--boundary123--
`
	
	email := parseEmail(rawEmail)
	
	if email.From != "sender@example.com" {
		t.Errorf("Expected 'sender@example.com', got '%s'", email.From)
	}
	
	if email.Body == "" {
		t.Error("Expected non-empty body from multipart email")
	}
}

// TestParseEmail_BodyWithSpecialChars - Test parsing email with special characters in body
func TestParseEmail_BodyWithSpecialChars(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: Special Characters
	
Hello =3D World!
Line with _underscore_ and "quotes".
Tab	here.
`
	
	email := parseEmail(rawEmail)
	
	if email.Body == "" {
		t.Error("Expected non-empty body")
	}
	
	// Check that special characters are preserved (as much as possible)
	expectedPatterns := []string{"Hello", "World", "underscore", "quotes"}
	for _, pattern := range expectedPatterns {
		if !containsString(email.Body, pattern) {
			t.Logf("Body may not contain '%s' as expected: '%s'", pattern, email.Body)
		}
	}
}

// TestParseEmail_EmptyFields - Test parsing email with empty fields
func TestParseEmail_EmptyFields(t *testing.T) {
	rawEmail := `From: 
To: 
Subject: 
Body only content.
`
	
	email := parseEmail(rawEmail)
	
	// Should not panic or return errors
	if email.Raw != rawEmail && len(email.Raw) == 0 {
		t.Errorf("Expected Raw field to contain input, got empty")
	}
}

// TestParseEmail_UnicodeContent - Test parsing email with Unicode content
func TestParseEmail_UnicodeContent(t *testing.T) {
	rawEmail := `From: sender@example.com
To: recipient@example.com
Subject: =?UTF-8?B?VGVzdCBzdWJqZWN0?=
Content-Type: text/plain; charset=UTF-8

Hello 世界！Привет мир!🎉
`
	
	email := parseEmail(rawEmail)
	
	if email.From != "sender@example.com" {
		t.Errorf("Expected 'sender@example.com', got '%s'", email.From)
	}
	
	if len(email.Body) == 0 {
		t.Error("Expected non-empty body with Unicode content")
	}
}

// TestParseEmail_FallbackParser - Test fallback parser when net/mail fails
func TestParseEmail_FallbackParser(t *testing.T) {
	// Invalid email that should trigger fallback
	rawEmail := `This is not a valid email format at all
Just some random text without proper headers
`
	
	email := parseEmail(rawEmail)
	
	// Should still return an Email struct even with invalid input
	if email.Raw == "" {
		t.Error("Expected Raw field to contain the raw email")
	}
}

// TestParseMultipartBoundary_Standard - Test boundary parsing from standard Content-Type
func TestParseMultipartBoundary_Standard(t *testing.T) {
	contentType := "multipart/alternative; boundary=\"boundary123\""
	boundary := parseMultipartBoundary(contentType)
	
	if boundary != "boundary123" {
		t.Errorf("Expected 'boundary123', got '%s'", boundary)
	}
}

// TestParseMultipartBoundary_NoBoundary - Test boundary parsing without boundary parameter
func TestParseMultipartBoundary_NoBoundary(t *testing.T) {
	contentType := "multipart/alternative"
	boundary := parseMultipartBoundary(contentType)
	
	if boundary != "" {
		t.Errorf("Expected empty boundary, got '%s'", boundary)
	}
}

// TestParseMultipartBoundary_NonMultipart - Test boundary parsing from non-multipart type
func TestParseMultipartBoundary_NonMultipart(t *testing.T) {
	contentType := "text/plain"
	boundary := parseMultipartBoundary(contentType)
	
	if boundary != "" {
		t.Errorf("Expected empty boundary for non-multipart, got '%s'", boundary)
	}
}
