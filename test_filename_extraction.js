// Test the filename extraction logic
function extractFilenameFromContentDisposition(contentDisposition) {
  // First, try to extract RFC 5987 format: filename*=UTF-8''encoded-filename
  const rfc5987Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/);
  if (rfc5987Match) {
    try {
      // URL decode the filename
      return decodeURIComponent(rfc5987Match[1]);
    } catch (e) {
      console.warn('Failed to decode RFC 5987 filename:', e);
      // Fall through to legacy format
    }
  }

  // Fallback to legacy format: filename="filename"
  const legacyMatch = contentDisposition.match(/filename="([^"]+)"/);
  if (legacyMatch) {
    return legacyMatch[1];
  }

  return null;
}

// Test cases
const testCases = [
  {
    name: "ASCII filename only",
    input: 'attachment; filename="document.pdf"',
    expected: "document.pdf"
  },
  {
    name: "Japanese filename with RFC 5987",
    input: 'attachment; filename="_____ (720 x 240 px).png"; filename*=UTF-8\'\'%E5%90%8D%E7%A7%B0%E6%9C%AA%E8%A8%AD%E5%AE%9A%20%28720%20x%20240%20px%29.png',
    expected: "名称未設定 (720 x 240 px).png"
  },
  {
    name: "Mixed characters with RFC 5987",
    input: 'attachment; filename="test-____.txt"; filename*=UTF-8\'\'test-%E3%83%95%E3%82%A1%E3%82%A4%E3%83%AB.txt',
    expected: "test-ファイル.txt"
  }
];

console.log("Testing filename extraction...\n");

testCases.forEach((testCase, index) => {
  const result = extractFilenameFromContentDisposition(testCase.input);
  const success = result === testCase.expected;
  
  console.log(`Test ${index + 1}: ${testCase.name}`);
  console.log(`Input: ${testCase.input}`);
  console.log(`Expected: "${testCase.expected}"`);
  console.log(`Result: "${result}"`);
  console.log(`Status: ${success ? '✅ PASS' : '❌ FAIL'}`);
  console.log('---');
});