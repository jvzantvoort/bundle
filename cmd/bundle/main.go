// Package main implements the bundle CLI tool for managing content-addressable
// file bundles with SHA256-based integrity verification.
//
// The CLI provides commands for creating, verifying, and managing bundles:
//
//	bundle create <path> --title "My Bundle"
//	bundle verify <path>
//	bundle info <path>
//	bundle list <path>
//	bundle tag add <path> <tag>...
//	bundle tag remove <path> <tag>...
//	bundle tag list <path>
//	bundle rename <path> <new_title>
//
// All commands support --json flag for machine-readable output and --verbose
// flag for detailed logging.
//
// Example usage:
//
//	# Create a bundle
//	$ bundle create ./photos --title "Vacation 2024"
//	Bundle Created
//	--------------
//	Path:     /home/user/photos
//	Checksum: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
//	Files:    42
//
//	# Verify integrity
//	$ bundle verify ./photos
//	Bundle Integrity: VALID
//
//	# Add tags
//	$ bundle tag add ./photos travel vacation europe
//
//	# Get JSON output
//	$ bundle info ./photos --json
//	{
//	  "path": "/home/user/photos",
//	  "title": "Vacation 2024",
//	  "files": 42,
//	  ...
//	}
package main

func main() {
	Execute()
}
