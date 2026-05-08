# Separate Dagger frontend build from GoReleaser release jobs

This is the document workspace for ticket dagger-frontend-ci.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket dagger-frontend-ci --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket dagger-frontend-ci --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket dagger-frontend-ci --field Status --value review`
