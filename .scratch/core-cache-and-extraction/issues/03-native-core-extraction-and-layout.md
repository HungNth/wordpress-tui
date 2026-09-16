# 03: Safe Native Core Zip Extraction and Standard Directory Layout

**What to build:** A self-contained native ZIP extractor for WordPress Core Archives that extracts the content-stripped archive into a target Website directory. It strips the leading `wordpress/` directory prefix from every entry, defends strictly against Zip Slip directory traversal vulnerabilities, and automatically scaffolds standard `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` directories with `0755` permissions.

**Blocked by:** 01: Core Cache Management and Manifest Storage

**Status:** resolved

- [x] Extracts all files and subdirectories from a Core Archive into the destination directory without invoking external tools
- [x] Strips the root `wordpress/` folder prefix from all archive entries so files reside directly at the website root
- [x] Rejects archive entries that attempt directory traversal or escape the destination root (Zip Slip protection)
- [x] Automatically creates `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` with `0755` permissions
- [x] Cleans up partially extracted files or directories if extraction errors midway
