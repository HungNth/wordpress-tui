package packages

import (
	"archive/zip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const MaxArtifactCap = 1024 * 1024 * 1024 // 1 GiB

// IsDisallowedIP checks for loopback, private (IPv4 RFC1918 and IPv6 ULA fc00::/7), link-local, unspecified, and cloud metadata IPs.
func IsDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// Cloud metadata IP: 169.254.169.254
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}

type DNSLookupFunc func(ctx context.Context, host string) ([]net.IP, error)
type RawDialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type DownloadOptions struct {
	LookupIP   DNSLookupFunc
	RawDial    RawDialFunc
	RootCAs    *x509.CertPool
	AllowedIPs []net.IP // Strictly for local test fixtures: only these exact IPs are exempted from SSRF filtering
}

// SafeDownload streams a remote HTTPS package into destPath enforcing size bounds, SSRF checks, and ZIP integrity.
func SafeDownload(ctx context.Context, downloadURL string, expectedSize int64, destPath string, optList ...DownloadOptions) error {
	var opt DownloadOptions
	if len(optList) > 0 {
		opt = optList[0]
	}

	u, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download URL must use HTTPS, got scheme %q", u.Scheme)
	}
	if expectedSize <= 0 || expectedSize > MaxArtifactCap {
		return fmt.Errorf("invalid download size %d (must be between 1 and 1 GiB)", expectedSize)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	partPath := destPath + ".part"
	partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create partial download file: %w", err)
	}

	cleanedUp := false
	defer func() {
		_ = partFile.Close()
		if !cleanedUp {
			_ = os.Remove(partPath)
		}
	}()

	lookupFn := opt.LookupIP
	if lookupFn == nil {
		lookupFn = func(ctx context.Context, host string) ([]net.IP, error) {
			return net.DefaultResolver.LookupIP(ctx, "ip", host)
		}
	}

	rawDialFn := opt.RawDial
	if rawDialFn == nil {
		defaultDialer := &net.Dialer{Timeout: 30 * time.Second}
		rawDialFn = defaultDialer.DialContext
	}

	pinnedDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := lookupFn(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed for %s: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses resolved for %s", host)
		}

		var pinnedIP net.IP
		for _, ip := range ips {
			isExplicitlyAllowed := false
			for _, allowed := range opt.AllowedIPs {
				if ip.Equal(allowed) {
					isExplicitlyAllowed = true
					break
				}
			}
			if !isExplicitlyAllowed && IsDisallowedIP(ip) {
				return nil, fmt.Errorf("SSRF protection: host %s resolved to disallowed IP address %s", host, ip.String())
			}
			if pinnedIP == nil {
				pinnedIP = ip
			}
		}

		return rawDialFn(ctx, network, net.JoinHostPort(pinnedIP.String(), port))
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Production TLS must always verify certificates
		RootCAs:            opt.RootCAs,
	}

	transport := &http.Transport{
		DialContext:     pinnedDialContext,
		TLSClientConfig: tlsConfig,
	}

	redirectCount := 0
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > 5 {
				return errors.New("stopped after 5 redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-HTTPS URL %s rejected", SanitizeURL(req.URL.String()))
			}
			req.Header.Del("Referer")
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed for %s: %w", SanitizeURL(downloadURL), SanitizeError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("download returned HTTP status %d for %s", resp.StatusCode, SanitizeURL(downloadURL))}
	}

	if resp.ContentLength > 0 && resp.ContentLength != expectedSize {
		return fmt.Errorf("server declared Content-Length %d does not match expected size %d", resp.ContentLength, expectedSize)
	}

	limitReader := io.LimitReader(resp.Body, expectedSize+1)
	written, err := io.Copy(partFile, limitReader)
	if err != nil {
		return fmt.Errorf("download streaming failed: %w", err)
	}

	if written > expectedSize {
		return fmt.Errorf("downloaded content exceeded expected size %d", expectedSize)
	}
	if written < expectedSize {
		return fmt.Errorf("download ended prematurely: received %d bytes, expected %d", written, expectedSize)
	}

	_ = partFile.Close()

	// Verify ZIP structure
	zipReader, err := zip.OpenReader(partPath)
	if err != nil {
		return fmt.Errorf("downloaded artifact is not a valid ZIP archive: %w", err)
	}
	_ = zipReader.Close()

	// Atomically move into destination path
	if err := os.Rename(partPath, destPath); err != nil {
		return fmt.Errorf("failed to commit downloaded file: %w", err)
	}

	cleanedUp = true
	return nil
}
