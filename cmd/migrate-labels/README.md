# Label Migration Tool

⚠️ **URGENT: Required Migration for v0.12.0+**

Starting with v0.12.0, the simple label format has been completely removed. All users **must** migrate to the extended format before upgrading. This tool helps you convert your existing labels.yaml file to the required format.

A CLI tool to migrate label configuration files from simple format to extended multi-label format.

## Features

- ✅ Convert simple label format to extended multi-label format
- ✅ Preserve cluster-wide access entries
- ✅ Skip already-migrated extended format entries
- ✅ Dry-run mode for preview
- ✅ Validation mode for analysis
- ✅ Detailed migration statistics

## Installation

```bash
go build -o migrate-labels ./cmd/migrate-labels
```

## Usage

### Basic Migration

Convert a simple labels.yaml to extended format:

```bash
./migrate-labels -input configs/labels.yaml
```

This creates `configs/labels-extended.yaml` with the converted format.

### Custom Output Path

Specify a custom output file:

```bash
./migrate-labels -input configs/labels.yaml -output configs/labels-new.yaml
```

### Custom Default Label

Specify the default label name for conversion (default: "namespace"):

```bash
./migrate-labels -input configs/labels.yaml -default-label "team"
```

### Dry Run (Preview)

Preview the conversion without writing files:

```bash
./migrate-labels -input configs/labels.yaml -dry-run
```

### Validation Mode

Analyze the file without conversion:

```bash
./migrate-labels -input configs/labels.yaml -validate
```

## Examples

### Input: Simple Format

```yaml
user1:
  hogarama: true
  prod: true

user2:
  dev: true

admin-group:
  '#cluster-wide': true
```

### Output: Extended Format

```yaml
user1:
  _rules:
    - name: namespace
      operator: "="
      values:
        - hogarama
        - prod
  _logic: AND

user2:
  _rules:
    - name: namespace
      operator: "="
      values:
        - dev
  _logic: AND

admin-group:
  '#cluster-wide': true  # Preserved as-is
```

## Migration Statistics

The tool provides detailed statistics:

```
--- Migration Statistics ---
Total entries:     3
Simple format:     2
Extended format:   0
Cluster-wide:      1
Converted:         2
Skipped:           1
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-input` | Path to input labels.yaml file (required) | - |
| `-output` | Path to output file | `<input>-extended.yaml` |
| `-default-label` | Default label name for conversion | `namespace` |
| `-dry-run` | Preview conversion without writing | `false` |
| `-validate` | Validate and analyze only | `false` |

## Pre-Upgrade Check

Before upgrading to v0.12.0+, verify if your labels.yaml needs migration:

```bash
./migrate-labels -input configs/labels.yaml -validate
```

**If the output shows "Simple format: 0"**, you're already using extended format and can upgrade safely.

**If the output shows any simple format entries**, you must complete the migration workflow below before upgrading.

## Migration Workflow

1. **Backup**: Create a backup of your current labels.yaml
   ```bash
   cp configs/labels.yaml configs/labels.yaml.backup
   ```

2. **Validate**: Analyze your current configuration
   ```bash
   ./migrate-labels -input configs/labels.yaml -validate
   ```

3. **Preview**: Check the conversion output
   ```bash
   ./migrate-labels -input configs/labels.yaml -dry-run
   ```

4. **Convert**: Generate the extended format
   ```bash
   ./migrate-labels -input configs/labels.yaml -output configs/labels-new.yaml
   ```

5. **Test**: Validate the new format works correctly
   ```bash
   # Update ConfigMap or mount new file
   # Test with actual queries
   ```

6. **Deploy**: Replace old format with new format
   ```bash
   mv configs/labels-new.yaml configs/labels.yaml
   # Redeploy or restart proxy
   ```

## Rollback

If issues occur, restore from backup:

```bash
cp configs/labels.yaml.backup configs/labels.yaml
# Redeploy or restart proxy
```

## Notes

- The tool preserves `#cluster-wide` entries without conversion
- Already extended format entries are skipped
- The default operator for simple format is `=` (equals)
- Multiple values are combined with OR logic using regex operator
- The tool does not modify the input file unless `-output` points to the same path
