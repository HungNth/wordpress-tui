package packages_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/packages"
)

func createValidZipBytes() []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, _ := w.Create("plugin.php")
	_, _ = f.Write([]byte("<?php // test plugin"))
	_ = w.Close()
	return buf.Bytes()
}

func TestBuildMetadataURL_PreservesPathPrefix(t *testing.T) {
	tests := []struct {
		baseURL    string
		slug       string
		key        string
		expected   string
		expectLeak bool
	}{
		{
			baseURL:  "https://example.com/api/v1",
			slug:     "my-plugin",
			key:      "secret_key_123",
			expected: "https://example.com/api/v1/package/my-plugin/metadata?license_key=secret_key_123",
		},
		{
			baseURL:  "https://example.com/api/v1/",
			slug:     "my-plugin",
			key:      "",
			expected: "https://example.com/api/v1/package/my-plugin/metadata", // no license_key param
		},
	}

	for _, tt := range tests {
		u, err := packages.BuildMetadataURL(tt.baseURL, tt.slug, tt.key)
		if err != nil {
			t.Fatalf("BuildMetadataURL failed: %v", err)
		}
		if u.String() != tt.expected {
			t.Errorf("BuildMetadataURL(%q, %q, %q) = %q, want %q", tt.baseURL, tt.slug, tt.key, u.String(), tt.expected)
		}
	}
}

func TestFetchMetadata_DoesNotLeakSecretInErrors(t *testing.T) {
	secretKey := "super_secret_license_999"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		// Server reflects key in error body
		_, _ = w.Write([]byte("Invalid key: " + r.URL.Query().Get("license_key")))
	}))
	defer server.Close()

	_, err := packages.FetchMetadata(context.Background(), server.Client(), server.URL, packages.PackageTypePlugin, "test-plugin", secretKey)
	if err == nil {
		t.Fatal("expected error on 401 status, got nil")
	}

	if strings.Contains(err.Error(), secretKey) {
		t.Fatalf("FetchMetadata leaked secret key in error string: %v", err)
	}
}

func TestSafeDownload_SuccessAndValidation(t *testing.T) {
	zipData := createValidZipBytes()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	destFile := filepath.Join(tempDir, "plugin.zip")

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(server.Certificate())

	serverIP := server.Listener.Addr().(*net.TCPAddr).IP
	opts := packages.DownloadOptions{
		AllowedIPs: []net.IP{serverIP}, // only allow exact local test server IP
		RootCAs:    rootCAs,
	}

	err := packages.SafeDownload(context.Background(), server.URL+"/download?license_key=secret&sig=123", int64(len(zipData)), destFile, opts)
	if err != nil {
		t.Fatalf("SafeDownload failed: %v", err)
	}

	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("dest file not found: %v", err)
	}
	if info.Size() != int64(len(zipData)) {
		t.Errorf("expected size %d, got %d", len(zipData), info.Size())
	}

	// Verify part file was cleaned up
	partFile := destFile + ".part"
	if _, err := os.Stat(partFile); !os.IsNotExist(err) {
		t.Errorf("expected part file to be removed")
	}
}

func TestSafeDownload_RejectsPrivateAndMetadataIPs(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"loopback ipv4", "127.0.0.1"},
		{"loopback ipv6", "::1"},
		{"private 10.x", "10.0.0.1"},
		{"private 192.168.x", "192.168.1.1"},
		{"private 172.16.x", "172.16.0.1"},
		{"ipv6 ULA fc00::/7", "fc00::1"},
		{"ipv6 ULA fd00::/8", "fd12:3456:789a:1::1"},
		{"cloud metadata ip", "169.254.169.254"},
		{"link-local ipv6", "fe80::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedIP := net.ParseIP(tt.ip)
			if !packages.IsDisallowedIP(parsedIP) {
				t.Errorf("expected IP %s to be disallowed", tt.ip)
			}
		})
	}
}

func TestSafeDownload_SSRFRejectionWithCustomResolver(t *testing.T) {
	opts := packages.DownloadOptions{
		LookupIP: func(ctx context.Context, host string) ([]net.IP, error) {
			// Mock DNS resolving to private IP
			return []net.IP{net.ParseIP("10.0.0.1")}, nil
		},
	}

	err := packages.SafeDownload(context.Background(), "https://malicious.example.com/package.zip", 1024, "/tmp/unused.zip", opts)
	if err == nil {
		t.Fatal("expected SSRF error on private IP, got nil")
	}
	if !strings.Contains(err.Error(), "SSRF protection") {
		t.Errorf("expected SSRF protection error message, got: %v", err)
	}
}

func TestSafeDownload_RedirectStripsRefererAndKeepsLocationOnly(t *testing.T) {
	zipData := createValidZipBytes()

	var redirectRequestReferer string
	var redirectRequestQuery string

	server2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectRequestReferer = r.Header.Get("Referer")
		redirectRequestQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer server2.Close()

	server1 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server2.URL+"/final-download?signed=true", http.StatusFound)
	}))
	defer server1.Close()

	tempDir := t.TempDir()
	destFile := filepath.Join(tempDir, "redirected.zip")

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(server1.Certificate())
	rootCAs.AddCert(server2.Certificate())

	ip1 := server1.Listener.Addr().(*net.TCPAddr).IP
	ip2 := server2.Listener.Addr().(*net.TCPAddr).IP

	opts := packages.DownloadOptions{
		AllowedIPs: []net.IP{ip1, ip2},
		RootCAs:    rootCAs,
	}

	initialURL := server1.URL + "/start?license_key=original_secret"
	err := packages.SafeDownload(context.Background(), initialURL, int64(len(zipData)), destFile, opts)
	if err != nil {
		t.Fatalf("SafeDownload failed on redirect: %v", err)
	}

	if redirectRequestReferer != "" {
		t.Errorf("expected Referer to be suppressed on redirect, got %q", redirectRequestReferer)
	}
	if strings.Contains(redirectRequestQuery, "license_key") {
		t.Errorf("original query with secret leaked into redirect request: %q", redirectRequestQuery)
	}
	if redirectRequestQuery != "signed=true" {
		t.Errorf("expected redirect to use Location query 'signed=true', got %q", redirectRequestQuery)
	}
}

func TestSafeDownload_RedactsSecretsInErrors(t *testing.T) {
	secretURL := "https://example.com/api/v1/download?license_key=super_secret_key&signature=abc12345"
	sanitized := packages.SanitizeURL(secretURL)

	if strings.Contains(sanitized, "super_secret_key") || strings.Contains(sanitized, "abc12345") {
		t.Errorf("SanitizeURL leaked query parameters: %s", sanitized)
	}
	if !strings.Contains(sanitized, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in sanitized URL: %s", sanitized)
	}
}

func TestSafeDownload_WrappedErrorSanitization(t *testing.T) {
	secretURL := "https://127.0.0.1:59999/download.zip?license_key=super_secret_12345&sig=abcdef"
	opts := packages.DownloadOptions{
		AllowedIPs: []net.IP{net.ParseIP("127.0.0.1")},
	}
	err := packages.SafeDownload(context.Background(), secretURL, 1024, filepath.Join(t.TempDir(), "f.zip"), opts)
	if err == nil {
		t.Fatal("expected download error, got nil")
	}

	errString := err.Error()
	if strings.Contains(errString, "super_secret_12345") || strings.Contains(errString, "sig=abcdef") {
		t.Fatalf("SafeDownload wrapped error leaked query secrets: %s", errString)
	}
	if !strings.Contains(errString, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in error string, got: %s", errString)
	}
}

func TestFetchMetadata_RejectsHostlessURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"name": "Hostless",
			"slug": "hostless-test",
			"type": "plugin",
			"version": "1.0.0",
			"size": "100",
			"download_url": "https:///missing-host.zip"
		}`))
	}))
	defer server.Close()
	_, err := packages.FetchMetadata(context.Background(), server.Client(), server.URL, packages.PackageTypePlugin, "hostless-test", "")
	if err == nil {
		t.Fatal("expected error on hostless download_url, got nil")
	}
}
