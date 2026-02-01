# Archiving Completed Changes

This guide covers Stage 3: Archiving changes after successful deployment.

## When to Archive

Archive a change after:

- The change has been deployed to production
- All features are verified working in production
- No rollback is anticipated

## Archive Workflow

After deployment, create a separate PR to archive the change:

### 1. Move Change Directory

Move the change directory from `changes/[name]/` to `changes/archive/YYYY-MM-DD-[name]/`:

```bash
# Example
mv openspec/changes/add-two-factor-auth openspec/changes/archive/2025-01-15-add-two-factor-auth
```

### 2. Update Specs

If the change added, modified, or removed capabilities:

- Merge spec deltas into `openspec/specs/[capability]/spec.md`
- Remove delta operation headers (## ADDED, ## MODIFIED, etc.)
- Ensure final spec reflects current production state
- For spec formatting details, see `agents/propose.md`

### 3. Use Archive Command

For tooling-only changes (no spec updates needed), use:

```bash
openspec archive <change-id> --skip-specs --yes
```

**Flags:**

- `--skip-specs` - Skip spec updates (for internal/tooling changes)
- `--yes` / `-y` - Skip confirmation prompts (for automation)

Always pass the change ID explicitly.

### 4. Validate Archive

After archiving, run validation to confirm everything is correct:

```bash
openspec validate --strict --no-interactive
```

Fix any validation errors before merging the archive PR.

## Archive PR Checklist

- [ ] Change moved to `changes/archive/YYYY-MM-DD-[name]/`
- [ ] Specs updated to reflect production state (if applicable)
- [ ] Validation passes (`openspec validate --strict --no-interactive`)
- [ ] PR describes what was deployed and when
- [ ] No active references to the archived change remain

## Examples

### Archiving with Spec Updates

```bash
# Manual archive with spec updates
mv openspec/changes/add-oauth openspec/changes/archive/2025-01-20-add-oauth

# Edit openspec/specs/auth/spec.md to merge deltas
# Remove ## ADDED Requirements headers, keep content

# Validate
openspec validate --strict --no-interactive
```

### Archiving Tooling-Only Change

```bash
# Automated archive for internal changes
openspec archive refactor-logger --skip-specs --yes

# Validate
openspec validate --strict --no-interactive
```

## After Archiving

- Verify archive PR is merged
- Confirm `openspec list` no longer shows the change
- Archived change remains in git history for reference
- Specs reflect current production capabilities
