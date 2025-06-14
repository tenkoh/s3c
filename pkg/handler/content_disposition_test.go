package handler

import (
	"strings"
	"testing"
)

func TestSetContentDisposition(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectContains []string // Check if result contains these parts
		notContains  []string   // Should not contain these
	}{
		{
			name:     "ASCII filename",
			filename: "document.pdf",
			expectContains: []string{
				`attachment`,
				`filename="document.pdf"`,
			},
			notContains: []string{`filename*=`}, // Should not have RFC5987 encoding for ASCII
		},
		{
			name:     "Japanese filename - should preserve original",
			filename: "名称未設定 (720 x 240 px).png",
			expectContains: []string{
				`attachment`,
				`filename*=UTF-8''`, // Must use RFC5987 for non-ASCII
			},
			notContains: []string{
				`=?UTF-8?Q?`, // Should NOT use RFC2047 MIME encoding
				`___`,        // Should not replace with underscores in final output
			},
		},
		{
			name:     "Mixed ASCII and Unicode",
			filename: "test-ファイル.txt",
			expectContains: []string{
				`attachment`,
				`filename*=UTF-8''`,
			},
			notContains: []string{`=?UTF-8?Q?`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setContentDisposition(tt.filename)
			t.Logf("Input: %q", tt.filename)
			t.Logf("Output: %q", result)
			
			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("setContentDisposition(%q) = %q, should contain %q", tt.filename, result, expected)
				}
			}
			
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("setContentDisposition(%q) = %q, should NOT contain %q", tt.filename, result, notExpected)
				}
			}
		})
	}
}

// Test what browsers actually see when downloading
func TestContentDispositionBrowserCompatibility(t *testing.T) {
	// This test demonstrates what different browsers should see
	testFilename := "名称未設定 (720 x 240 px).png"
	result := setContentDisposition(testFilename)
	
	t.Logf("Japanese filename: %q", testFilename)
	t.Logf("Content-Disposition: %q", result)
	
	// The result should allow browsers to download with the original Japanese filename
	// Most modern browsers support RFC 5987 (filename*=UTF-8'') 
	// and will use that over the fallback filename=""
}