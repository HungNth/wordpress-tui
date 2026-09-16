# Dynamic Package size verification

Remote package download streams and cache hits verify integrity through valid ZIP structure, HTTP Content-Length matching, and version equality rather than static metadata size matching. Packages are generated or stamped dynamically on the distribution server with user license keys, causing archive byte counts to differ between metadata and the downloaded payload; downloads enforce a 1 GiB ceiling to prevent resource exhaustion while caching and stale fallbacks index and verify solely by package version.
