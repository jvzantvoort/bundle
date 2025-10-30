# Documentation Navigation Guide

Quick reference for finding information in the Bundle Library documentation.

---

## By Topic

### Getting Started
- **Installation** → [../README.md](../README.md#installation)
- **Quick Start** → [../README.md](../README.md#quick-start)
- **First Bundle** → [EXAMPLES.md](EXAMPLES.md#creating-bundles)

### Core Operations
- **Create Bundle** → [EXAMPLES.md](EXAMPLES.md#creating-bundles)
- **Verify Integrity** → [EXAMPLES.md](EXAMPLES.md#verifying-bundles)
- **View Information** → [EXAMPLES.md](EXAMPLES.md#bundle-info)
- **List Files** → [EXAMPLES.md](EXAMPLES.md#listing-files)

### Organization
- **Add Tags** → [EXAMPLES.md](EXAMPLES.md#tagging-bundles)
- **Update Title** → [RENAME_UPDATE.md](RENAME_UPDATE.md)
- **Pool Storage** → [POOLS.md](POOLS.md)

### Centralized Storage
- **Configure Pools** → [POOLS.md](POOLS.md#configuration)
- **Import Bundles** → [POOLS.md](POOLS.md#importing-bundles)
- **List Pool Contents** → [POOLS.md](POOLS.md#listing-bundles)
- **Content Deduplication** → [POOLS.md](POOLS.md#deduplication)

### Configuration
- **Config Files** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#configuration-files)
- **Pool Setup** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#pool-configuration)
- **Debug Logging** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#debug-logging)

### Troubleshooting
- **Common Issues** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#troubleshooting)
- **Error Messages** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#common-errors)
- **Debug Mode** → [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md#enabling-debug-logs)

### Development
- **API Reference** → [API.md](API.md)
- **Architecture** → [POOLS_IMPLEMENTATION.md](POOLS_IMPLEMENTATION.md)
- **Code Quality** → [GOLANGCI_LINT_FIXES.md](GOLANGCI_LINT_FIXES.md)
- **Specifications** → [../specs/001-bundle-core/](../specs/001-bundle-core/)

---

## By Role

### End User
1. [Quick Start Guide](../README.md#quick-start)
2. [Examples](EXAMPLES.md)
3. [Pool Management](POOLS.md)
4. [Configuration](CONFIG_DEBUGGING.md)

### System Administrator
1. [Pool Configuration](POOLS.md#configuration)
2. [Multi-Pool Setup](POOLS.md#multiple-pools)
3. [Debug Logging](CONFIG_DEBUGGING.md#debug-logging)
4. [Troubleshooting](CONFIG_DEBUGGING.md#troubleshooting)

### Developer
1. [API Documentation](API.md)
2. [Architecture Overview](POOLS_IMPLEMENTATION.md)
3. [Specifications](../specs/001-bundle-core/)
4. [Code Quality](GOLANGCI_LINT_FIXES.md)

### Project Manager
1. [Project Summary](SUMMARY.md)
2. [Specification Updates](SPEC_UPDATES.md)
3. [Future Improvements](IMPROVEMENTS.md)
4. [Analysis](ANALYSIS.md)

---

## By Task

### "I want to..."

#### Create and Manage Bundles
- **Create from directory** → [EXAMPLES.md#creating-bundles](EXAMPLES.md#creating-bundles)
- **Add title** → [EXAMPLES.md#creating-with-title](EXAMPLES.md#creating-with-title)
- **Update title** → [RENAME_UPDATE.md](RENAME_UPDATE.md)
- **Add tags** → [EXAMPLES.md#tagging-bundles](EXAMPLES.md#tagging-bundles)

#### Verify and Inspect
- **Check integrity** → [EXAMPLES.md#verifying-bundles](EXAMPLES.md#verifying-bundles)
- **View metadata** → [EXAMPLES.md#bundle-info](EXAMPLES.md#bundle-info)
- **List files** → [EXAMPLES.md#listing-files](EXAMPLES.md#listing-files)
- **Get JSON output** → [EXAMPLES.md#json-output](EXAMPLES.md#json-output)

#### Use Pools
- **Set up pools** → [POOLS.md#configuration](POOLS.md#configuration)
- **Import bundle** → [POOLS.md#importing-bundles](POOLS.md#importing-bundles)
- **List pool contents** → [POOLS.md#listing-bundles](POOLS.md#listing-bundles)
- **Move vs copy** → [POOLS.md#move-vs-copy](POOLS.md#move-vs-copy)

#### Configure System
- **Create config file** → [CONFIG_DEBUGGING.md#configuration-files](CONFIG_DEBUGGING.md#configuration-files)
- **Define pools** → [CONFIG_DEBUGGING.md#pool-configuration](CONFIG_DEBUGGING.md#pool-configuration)
- **Enable debug logs** → [CONFIG_DEBUGGING.md#debug-logging](CONFIG_DEBUGGING.md#debug-logging)
- **Trace config loading** → [CONFIG_DEBUGGING.md#configuration-tracing](CONFIG_DEBUGGING.md#configuration-tracing)

#### Develop with API
- **Use bundle package** → [API.md#package-bundle](API.md#package-bundle)
- **Work with metadata** → [API.md#package-metadata](API.md#package-metadata)
- **Compute checksums** → [API.md#package-checksum](API.md#package-checksum)
- **Manage pools** → [API.md#package-pool](API.md#package-pool)

#### Understand Implementation
- **Architecture** → [POOLS_IMPLEMENTATION.md#architecture](POOLS_IMPLEMENTATION.md#architecture)
- **Design decisions** → [POOLS_IMPLEMENTATION.md#design-decisions](POOLS_IMPLEMENTATION.md#design-decisions)
- **Code structure** → [POOLS_IMPLEMENTATION.md#code-structure](POOLS_IMPLEMENTATION.md#code-structure)
- **Testing approach** → [POOLS_IMPLEMENTATION.md#testing](POOLS_IMPLEMENTATION.md#testing)

#### Troubleshoot Issues
- **Config not found** → [CONFIG_DEBUGGING.md#config-file-not-found](CONFIG_DEBUGGING.md#config-file-not-found)
- **Pool errors** → [CONFIG_DEBUGGING.md#pool-errors](CONFIG_DEBUGGING.md#pool-errors)
- **Integrity failures** → [EXAMPLES.md#handling-corruption](EXAMPLES.md#handling-corruption)
- **Debug mode** → [CONFIG_DEBUGGING.md#enabling-debug-logs](CONFIG_DEBUGGING.md#enabling-debug-logs)

---

## Document Descriptions

### User Guides

#### [EXAMPLES.md](EXAMPLES.md) (15KB)
Comprehensive examples of all bundle operations:
- Creating and verifying bundles
- Managing tags
- Using JSON output
- Error handling
- Integration workflows

#### [POOLS.md](POOLS.md) (8KB)
Complete guide to pool management:
- Configuration format
- Importing bundles
- Listing pool contents
- Content-addressable storage
- Deduplication

#### [RENAME_UPDATE.md](RENAME_UPDATE.md) (7KB)
Bundle title management:
- Rename command usage
- Preserving checksums
- CLI and API examples
- Testing information

#### [CONFIG_DEBUGGING.md](CONFIG_DEBUGGING.md) (9KB)
Configuration and troubleshooting:
- Config file locations
- Debug logging setup
- Common issues
- Configuration tracing

### Developer Guides

#### [API.md](API.md) (16KB)
Complete API reference:
- All package documentation
- Function signatures
- Data structures
- Usage examples

#### [POOLS_IMPLEMENTATION.md](POOLS_IMPLEMENTATION.md) (8KB)
Technical implementation:
- Architecture overview
- Design decisions
- Code organization
- Testing approach

#### [GOLANGCI_LINT_FIXES.md](GOLANGCI_LINT_FIXES.md) (6KB)
Code quality guide:
- Linting fixes applied
- Error handling patterns
- Security improvements
- Performance optimizations

### Project Documentation

#### [SPEC_UPDATES.md](SPEC_UPDATES.md) (7KB)
Recent specification changes:
- New commands
- Data model updates
- Requirement changes
- Compliance status

#### [SUMMARY.md](SUMMARY.md) (5KB)
Project overview:
- Feature summary
- Implementation status
- Testing coverage
- Next steps

#### [ANALYSIS.md](ANALYSIS.md) (5KB)
Code analysis:
- Package structure
- Code metrics
- Dependencies
- Coverage statistics

#### [IMPROVEMENTS.md](IMPROVEMENTS.md) (6KB)
Future enhancements:
- Planned features
- Optimization opportunities
- Architecture improvements
- Community requests

#### [DOCUMENTATION.md](DOCUMENTATION.md) (5KB)
Documentation standards:
- Writing guidelines
- Structure conventions
- Example format

#### [COPILOT.md](COPILOT.md) (6KB)
AI-assisted development notes:
- Code generation approach
- Review process
- Quality assurance

---

## File Organization

```
docs/
├── README.md                    # Documentation index (this file)
├── NAVIGATION.md                # This navigation guide
│
├── User Guides/
│   ├── EXAMPLES.md              # Usage examples
│   ├── POOLS.md                 # Pool management
│   ├── RENAME_UPDATE.md         # Rename command
│   └── CONFIG_DEBUGGING.md      # Configuration
│
├── Developer Guides/
│   ├── API.md                   # API reference
│   ├── POOLS_IMPLEMENTATION.md  # Implementation
│   └── GOLANGCI_LINT_FIXES.md   # Code quality
│
└── Project Documentation/
    ├── SPEC_UPDATES.md          # Spec changes
    ├── SUMMARY.md               # Project summary
    ├── ANALYSIS.md              # Code analysis
    ├── IMPROVEMENTS.md          # Roadmap
    ├── DOCUMENTATION.md         # Doc standards
    └── COPILOT.md               # AI notes
```

---

## External References

### Specifications
- [Feature Spec](../specs/001-bundle-core/spec.md)
- [Data Model](../specs/001-bundle-core/data-model.md)
- [CLI Commands](../specs/001-bundle-core/contracts/cli-commands.md)
- [Implementation Plan](../specs/001-bundle-core/plan.md)

### Main Project
- [README](../README.md)
- [LICENSE](../LICENSE)
- [Example Config](../config.yaml.example)

---

## Quick Command Reference

```bash
# Create bundle
bundle create ./photos --title "Vacation"

# Verify
bundle verify ./photos

# Info
bundle info ./photos

# Tags
bundle tag add ./photos travel 2024
bundle tag list ./photos

# Rename
bundle rename ./photos "Updated Title"

# Import to pool
bundle import ./photos --pool default

# List pool
bundle list_bundles --pool default

# JSON output
bundle info ./photos --json
```

For complete command reference, see [CLI Commands](../specs/001-bundle-core/contracts/cli-commands.md).

---

## Search Tips

### By Keyword

- **"create"** → EXAMPLES.md, API.md
- **"verify"** → EXAMPLES.md, API.md
- **"pool"** → POOLS.md, POOLS_IMPLEMENTATION.md
- **"config"** → CONFIG_DEBUGGING.md, POOLS.md
- **"tag"** → EXAMPLES.md, API.md
- **"rename"** → RENAME_UPDATE.md, API.md
- **"import"** → POOLS.md, POOLS_IMPLEMENTATION.md
- **"debug"** → CONFIG_DEBUGGING.md
- **"error"** → CONFIG_DEBUGGING.md, GOLANGCI_LINT_FIXES.md
- **"API"** → API.md, ../specs/
- **"spec"** → SPEC_UPDATES.md, ../specs/

### By File Type

- **Command usage** → EXAMPLES.md
- **Configuration** → CONFIG_DEBUGGING.md, POOLS.md
- **Code examples** → API.md, EXAMPLES.md
- **Architecture** → POOLS_IMPLEMENTATION.md
- **Troubleshooting** → CONFIG_DEBUGGING.md
- **Project status** → SUMMARY.md, SPEC_UPDATES.md

---

**Last Updated**: 2025-10-30  
**Documentation Version**: 1.0.0
