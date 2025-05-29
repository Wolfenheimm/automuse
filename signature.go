package main

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
)

// SignatureParser handles YouTube signature parsing
type SignatureParser struct {
	transformMap map[string]string
}

// NewSignatureParser creates a new signature parser
func NewSignatureParser() *SignatureParser {
	return &SignatureParser{
		transformMap: make(map[string]string),
	}
}

// ParseSignature parses the signature from a cipher string
func (sp *SignatureParser) ParseSignature(cipher string) (string, error) {
	// Extract the signature from the cipher string
	sig := extractSignature(cipher)
	if sig == "" {
		return "", nil
	}

	// Extract the transform functions from the player source
	if err := sp.extractTransformFunctions(); err != nil {
		return "", err
	}

	// Apply the transformations to the signature
	transformedSig := sp.transformSignature(sig)
	return transformedSig, nil
}

// extractSignature extracts the signature from a cipher string
func extractSignature(cipher string) string {
	// The signature is usually in the format s=<signature>
	re := regexp.MustCompile(`s=([^&]+)`)
	matches := re.FindStringSubmatch(cipher)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractTransformFunctions extracts the transform functions from the player source
func (sp *SignatureParser) extractTransformFunctions() error {
	// This is a simplified version - in reality, we need to:
	// 1. Download the player source
	// 2. Extract the transform functions
	// 3. Parse them into a map of function names to their implementations

	// For now, we'll use some common transformations
	sp.transformMap = map[string]string{
		"reverse": "function(a){a.reverse()}",
		"swap":    "function(a,b){var c=a[0];a[0]=a[b%a.length];a[b%a.length]=c}",
		"splice":  "function(a,b){a.splice(0,b)}",
	}

	return nil
}

// transformSignature applies the transformations to the signature
func (sp *SignatureParser) transformSignature(sig string) string {
	// Convert the signature to a slice of characters
	sigChars := strings.Split(sig, "")

	// Apply the transformations
	// Note: This is a simplified version. In reality, we need to:
	// 1. Parse the transform functions properly
	// 2. Apply them in the correct order
	// 3. Handle different types of transformations

	// For now, we'll just reverse the signature as a basic example
	for i, j := 0, len(sigChars)-1; i < j; i, j = i+1, j-1 {
		sigChars[i], sigChars[j] = sigChars[j], sigChars[i]
	}

	return strings.Join(sigChars, "")
}

// getStreamURLWithSignature gets a stream URL with a properly transformed signature
func getStreamURLWithSignature(format *YouTubeFormat) (string, error) {
	parser := NewSignatureParser()

	// Extract the cipher from the format
	cipher := format.Cipher
	if cipher == "" {
		// If there's no cipher, the URL is already good
		return format.URL, nil
	}

	// Parse the signature
	sig, err := parser.ParseSignature(cipher)
	if err != nil {
		return "", err
	}

	// Extract the base URL and parameters from the cipher
	re := regexp.MustCompile(`url=([^&]+)`)
	matches := re.FindStringSubmatch(cipher)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract URL from cipher")
	}

	// URL decode the base URL
	baseURL, err := url.QueryUnescape(matches[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode URL: %v", err)
	}

	// Construct the final URL with all parameters
	url := baseURL
	if sig != "" {
		// First try to replace the signature parameter
		url = strings.Replace(url, "signature=", "signature="+sig, 1)

		// If that didn't work, try to append it
		if url == baseURL {
			if strings.Contains(url, "?") {
				url += "&signature=" + sig
			} else {
				url += "?signature=" + sig
			}
		}
	}

	// Log the URL construction process
	log.Printf("[DEBUG] Original URL: %s", format.URL)
	log.Printf("[DEBUG] Cipher: %s", cipher)
	log.Printf("[DEBUG] Base URL: %s", baseURL)
	log.Printf("[DEBUG] Transformed signature: %s", sig)
	log.Printf("[DEBUG] Final URL: %s", url)

	return url, nil
}
