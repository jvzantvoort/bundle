# Bundle Library Documentation

Complete documentation for the Bundle Library - a content-addressable digital asset management system.

---

## 📚 Table of Contents

### Getting Started
- [Quick Start Guide](../README.md) - Installation and basic usage
- [Examples](EXAMPLES.md) - Common usage patterns and workflows

### User Guides
- [Pool Management](POOLS.md) - Working with centralized bundle storage
- [Configuration & Debugging](CONFIG_DEBUGGING.md) - Configuration and troubleshooting

### Feature Documentation
- [Bundle Rename](RENAME_UPDATE.md) - Updating bundle titles
- [Import & Export](POOLS.md#importing-bundles) - Moving bundles to/from pools

### Developer Documentation
- [Pool Implementation](POOLS_IMPLEMENTATION.md) - Pool architecture and design
- [Code Quality](GOLANGCI_LINT_FIXES.md) - Linting and best practices
- [API Documentation](API.md) - Complete API reference

### Project Documentation
- [Specification Updates](SPEC_UPDATES.md) - Recent specification changes
- [Development Summary](SUMMARY.md) - Project overview and status
- [Analysis](ANALYSIS.md) - Code analysis and metrics
- [Improvements](IMPROVEMENTS.md) - Future enhancements

---

## 🎯 Quick Navigation

### For Users

**I want to...**

- **Create a bundle** → See [Examples: Creating Bundles](EXAMPLES.md#creating-bundles)
- **Verify integrity** → See [Examples: Verification](EXAMPLES.md#verifying-bundles)
- **Organize with pools** → See [Pool Management](POOLS.md)
- **Update bundle title** → See [Bundle Rename](RENAME_UPDATE.md)
- **Import to central storage** → See [Pools: Importing](POOLS.md#importing-bundles)
- **List all bundles** → See [Pools: Listing](POOLS.md#listing-bundles)
- **Add tags** → See [Examples: Tagging](EXAMPLES.md#tagging-bundles)
- **Troubleshoot** → See [Configuration & Debugging](CONFIG_DEBUGGING.md)

### For Developers

**I want to...**

- **Understand architecture** → See [Pool Implementation](POOLS_IMPLEMENTATION.md)
- **Add new features** → See [Improvements](IMPROVEMENTS.md)
- **Fix linting issues** → See [Code Quality](GOLANGCI_LINT_FIXES.md)
- **Review specifications** → See [Spec Updates](SPEC_UPDATES.md)
- **Understand APIs** → See [API Documentation](API.md)

---

## 📖 Documentation Structure

### User-Facing Documentation

#### [EXAMPLES.md](EXAMPLES.md)
Complete guide to common workflows:
- Bundle creation and verification
- Tagging and organization
- Pool operations
- JSON output usage
- Error handling

#### [POOLS.md](POOLS.md)
Centralized storage management:
- Pool configuration
- Importing bundles (copy/move)
- Listing pool contents
- Content-addressable storage
- Deduplication

#### [RENAME_UPDATE.md](RENAME_UPDATE.md)
Bundle metadata management:
- Renaming bundles
- Preserving checksums
- Command usage
- API reference

#### [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md)
Configuration and troubleshooting:
- Configuration file locations
- Debug logging
- Common issues
- Configuration tracing

### Developer Documentation

#### [POOLS_IMPLEMENTATION.md](POOLS_IMPLEMENTATION.md)
Technical implementation details:
- Architecture overview
- Design decisions
- Code structure
- Testing approach

#### [GOLANGCI_LINT_FIXES.md](GOLANGCI_LINT_FIXES.md)
Code quality and best practices:
- Linting fixes applied
- Error handling patterns
- Security improvements
- Performance optimizations

#### [API.md](API.md)
Complete API reference:
- Package documentation
- Function signatures
- Data structures
- Usage examples

### Project Documentation

#### [SPEC_UPDATES.md](SPEC_UPDATES.md)
Recent specification changes:
- New commands added
- Data model updates
- Requirement changes
- Compliance status

#### [SUMMARY.md](SUMMARY.md)
Project overview:
- Feature summary
- Implementation status
- Testing coverage
- Next steps

#### [ANALYSIS.md](ANALYSIS.md)
Code analysis:
- Package structure
- Code metrics
- Dependencies
- Coverage statistics

#### [IMPROVEMENTS.md](IMPROVEMENTS.md)
Future enhancements:
- Planned features
- Optimization opportunities
- Architecture improvements
- Community requests

---

## 🏗️ Architecture Overview

```
Bundle Library
├── Core Features
│   ├── Content-addressable bundles
│   ├── SHA256 checksums
│   ├── Integrity verification
│   └── Metadata management
│
├── Pool Management
│   ├── Centralized storage
│   ├── Content deduplication
│   ├── Import/export operations
│   └── Multi-pool support
│
├── CLI Interface
│   ├── create, verify, info, list
│   ├── tag (add, remove, list)
│   ├── rename, import, list_bundles
│   └── JSON output support
│
└── Configuration
    ├── YAML-based config
    ├── Multiple pool definitions
    ├── Debug logging
    └── Flexible search paths
```

---

## 🚀 Quick Start

### 1. Create a Bundle
```bash
bundle create ./my-photos --title "Vacation Photos 2024"
```

### 2. Add Tags
```bash
bundle tag add ./my-photos vacation travel iceland
```

### 3. Verify Integrity
```bash
bundle verify ./my-photos
```

### 4. Import to Pool
```bash
bundle import ./my-photos --pool default
```

### 5. List Pool Contents
```bash
bundle list_bundles --pool default
```

For more details, see [Examples](EXAMPLES.md).

---

## 📋 Command Reference

| Command | Description | Documentation |
|---------|-------------|---------------|
| `create` | Create new bundle | [Examples](EXAMPLES.md#creating-bundles) |
| `verify` | Check integrity | [Examples](EXAMPLES.md#verifying-bundles) |
| `info` | Display metadata | [Examples](EXAMPLES.md#bundle-info) |
| `list` | List bundle files | [Examples](EXAMPLES.md#listing-files) |
| `tag add` | Add tags | [Examples](EXAMPLES.md#tagging-bundles) |
| `tag remove` | Remove tags | [Examples](EXAMPLES.md#tagging-bundles) |
| `tag list` | List tags | [Examples](EXAMPLES.md#tagging-bundles) |
| `rename` | Update title | [Rename Guide](RENAME_UPDATE.md) |
| `import` | Import to pool | [Pools](POOLS.md#importing-bundles) |
| `list_bundles` | List pool bundles | [Pools](POOLS.md#listing-bundles) |

---

## 🔧 Configuration

### Configuration File Locations
1. `./config.yaml` (current directory)
2. `~/.config/bundle/config.yaml` (user config)
3. `/etc/bundle/config.yaml` (system config)

### Example Configuration
```yaml
pools:
  default:
    root: /mnt/bundles
    title: Default Bundle Pool
  
  backup:
    root: /backup/bundles
    title: Backup Pool

log_level: info
```

See [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md) for details.

---

## 🧪 Testing

### Run Tests
```bash
go test ./...
```

### Build
```bash
./build.sh
```

### Lint
```bash
golangci-lint run
```

See [GOLANGCI_LINT_FIXES.md](GOLANGCI_LINT_FIXES.md) for code quality details.

---

## 📊 Project Status

### Implementation Status
- ✅ Core bundle operations (create, verify, info, list)
- ✅ Tag management (add, remove, list)
- ✅ Pool management (import, list_bundles)
- ✅ Bundle rename (title updates)
- ✅ Configuration system
- ✅ Debug logging
- ✅ JSON output support

### Test Coverage
- ✅ 17/17 tests passing
- ✅ 100% specification compliance
- ✅ 0 linting issues

See [SUMMARY.md](SUMMARY.md) for complete status.

---

## 🔮 Future Work

### Planned Features
- Bundle search by tags/metadata
- Export from pools
- Remote replication (S3, SSH)
- Bundle rebuild (regenerate checksums)
- Web API interface
- Cryptographic signatures

See [IMPROVEMENTS.md](IMPROVEMENTS.md) for detailed roadmap.

---

## 📝 Specifications

Formal specifications are located in `../specs/001-bundle-core/`:

- [spec.md](../specs/001-bundle-core/spec.md) - User stories and requirements
- [data-model.md](../specs/001-bundle-core/data-model.md) - Data structures
- [cli-commands.md](../specs/001-bundle-core/contracts/cli-commands.md) - Command contracts

See [SPEC_UPDATES.md](SPEC_UPDATES.md) for recent changes.

---

## 🤝 Contributing

### Documentation Guidelines
- Keep examples current and tested
- Document all public APIs
- Include both table and JSON output examples
- Add troubleshooting for common issues

### Code Guidelines
- Follow golangci-lint rules
- Write comprehensive tests
- Add structured logging
- Update specifications

---

## 📄 License

See [LICENSE](../LICENSE) in the repository root.

---

## 📞 Support

For issues, questions, or contributions:
- Review [Examples](EXAMPLES.md) for common patterns
- Check [Troubleshooting](CONFIG_DEBUGGING.md#troubleshooting)
- See [Analysis](ANALYSIS.md) for code structure

---

**Last Updated**: 2025-10-30  
**Version**: 1.0.0  
**Status**: Production Ready
